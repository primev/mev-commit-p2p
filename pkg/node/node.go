package node

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	builderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/builderapi/v1"
	searcherapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/searcherapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/apiserver"
	"github.com/primevprotocol/mev-commit/pkg/debugapi"
	"github.com/primevprotocol/mev-commit/pkg/discovery"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p"
	"github.com/primevprotocol/mev-commit/pkg/preconfirmation"
	"github.com/primevprotocol/mev-commit/pkg/register"
	builderapi "github.com/primevprotocol/mev-commit/pkg/rpc/builder"
	searcherapi "github.com/primevprotocol/mev-commit/pkg/rpc/searcher"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
	"github.com/primevprotocol/mev-commit/pkg/topology"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type Options struct {
	Version          string
	PrivKey          *ecdsa.PrivateKey
	Secret           string
	PeerType         string
	Logger           *slog.Logger
	P2PPort          int
	HTTPPort         int
	RPCPort          int
	Bootnodes        []string
	ExposeBuilderAPI bool
}

type Node struct {
	closers []io.Closer
}

func NewNode(opts *Options) (*Node, error) {
	reg := register.New()

	minStake, err := reg.GetMinimumStake()
	if err != nil {
		return nil, err
	}

	nd := &Node{
		closers: make([]io.Closer, 0),
	}

	srv := apiserver.New(opts.Version, opts.Logger.With("component", "apiserver"))

	closers := make([]io.Closer, 0)
	peerType := p2p.FromString(opts.PeerType)

	p2pSvc, err := libp2p.New(&libp2p.Options{
		PrivKey:        opts.PrivKey,
		Secret:         opts.Secret,
		PeerType:       peerType,
		Register:       reg,
		MinimumStake:   minStake,
		Logger:         opts.Logger.With("component", "p2p"),
		ListenPort:     opts.P2PPort,
		MetricsReg:     srv.MetricsRegistry(),
		BootstrapAddrs: opts.Bootnodes,
	})
	if err != nil {
		return nil, err
	}
	nd.closers = append(nd.closers, p2pSvc)

	topo := topology.New(p2pSvc, opts.Logger.With("component", "topology"))
	disc := discovery.New(topo, p2pSvc, opts.Logger.With("component", "discovery_protocol"))
	nd.closers = append(nd.closers, disc)

	// Set the announcer for the topology service
	topo.SetAnnouncer(disc)
	// Set the notifier for the p2p service
	p2pSvc.SetNotifier(topo)

	// Register the discovery protocol with the p2p service
	p2pSvc.AddProtocol(disc.Protocol())

	debugapi.RegisterAPI(srv, topo, p2pSvc, opts.Logger.With("component", "debugapi"))

	if opts.PeerType != p2p.PeerTypeBootnode.String() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", opts.RPCPort))
		if err != nil {
			_ = nd.Close()
			return nil, err
		}
		grpcServer := grpc.NewServer()

		preconfSigner := preconfsigner.NewSigner(opts.PrivKey)
		var bidProcessor preconfirmation.BidProcessor = noOpBidProcessor{}

		switch opts.PeerType {
		case p2p.PeerTypeBuilder.String():
			if opts.ExposeBuilderAPI {
				builderAPI := builderapi.NewService(opts.Logger.With("component", "builderapi"))
				builderapiv1.RegisterBuilderServer(grpcServer, builderAPI)
				bidProcessor = builderAPI
			}
			// TODO(@ckartik): Update noOpBidProcessor to be selected as default in a flag paramater.
			preconfProto := preconfirmation.New(
				topo,
				p2pSvc,
				preconfSigner,
				noOpUserStore{},
				bidProcessor,
				opts.Logger.With("component", "preconfirmation_protocol"),
			)
			// Only register handler for builder
			p2pSvc.AddProtocol(preconfProto.Protocol())

		case p2p.PeerTypeSearcher.String():
			preconfProto := preconfirmation.New(
				topo,
				p2pSvc,
				preconfSigner,
				noOpUserStore{},
				bidProcessor,
				opts.Logger.With("component", "preconfirmation_protocol"),
			)

			searcherAPI := searcherapi.NewService(
				preconfProto,
				opts.Logger.With("component", "searcherapi"),
			)
			searcherapiv1.RegisterSearcherServer(grpcServer, searcherAPI)
		}

		started := make(chan struct{})
		go func() {
			// signal that the server has started
			close(started)

			err := grpcServer.Serve(lis)
			if err != nil {
				opts.Logger.Error("failed to start grpc server", "err", err)
			}
		}()
		nd.closers = append(nd.closers, lis)

		// Wait for the server to start
		<-started

		gwMux := runtime.NewServeMux()
		bgCtx := context.Background()

		grpcConn, err := grpc.DialContext(
			bgCtx,
			fmt.Sprintf(":%d", opts.RPCPort),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			opts.Logger.Error("failed to dial grpc server", "err", err)
			_ = nd.Close()
			return nil, err
		}

		switch opts.PeerType {
		case p2p.PeerTypeBuilder.String():
			err := builderapiv1.RegisterBuilderHandler(bgCtx, gwMux, grpcConn)
			if err != nil {
				opts.Logger.Error("failed to register builder handler", "err", err)
				return nil, err
			}
		case p2p.PeerTypeSearcher.String():
			err := searcherapiv1.RegisterSearcherHandler(bgCtx, gwMux, grpcConn)
			if err != nil {
				opts.Logger.Error("failed to register searcher handler", "err", err)
				return nil, err
			}
		}

		srv.ChainHandlers("/", gwMux)
		srv.ChainHandlers(
			"/health",
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					if s := grpcConn.GetState(); s != connectivity.Ready {
						http.Error(w, fmt.Sprintf("grpc server is %s", s), http.StatusBadGateway)
						return
					}
					fmt.Fprintln(w, "ok")
				},
			),
		)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", opts.HTTPPort),
		Handler: srv.Router(),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			opts.Logger.Error("failed to start server", "err", err)
		}
	}()
	closers = append(closers, server)

	return &Node{closers: closers}, nil
}

func (n *Node) Close() error {
	var err error
	for _, c := range n.closers {
		err = errors.Join(err, c.Close())
	}

	return err
}

type noOpUserStore struct{}

func (noOpUserStore) CheckUserRegistred(_ *common.Address) bool {
	return true
}

type noOpBidProcessor struct{}

// The noOpBidProcesor auto accepts all bids sent
func (noOpBidProcessor) ProcessBid(
	_ context.Context,
	_ *preconfsigner.Bid,
) (chan builderapiv1.BidResponse_Status, error) {
	statusC := make(chan builderapiv1.BidResponse_Status, 5)
	statusC <- builderapiv1.BidResponse_STATUS_ACCEPTED
	close(statusC)

	return statusC, nil
}
