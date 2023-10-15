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
	BuilderExposeAPI bool
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
	closers = append(closers, p2pSvc)

	topo := topology.New(p2pSvc, opts.Logger.With("component", "topology"))
	disc := discovery.New(topo, p2pSvc, opts.Logger.With("component", "discovery_protocol"))
	closers = append(closers, disc)

	// Set the announcer for the topology service
	topo.SetAnnouncer(disc)
	// Set the notifier for the p2p service
	p2pSvc.SetNotifier(topo)

	// Register the discovery protocol with the p2p service
	p2pSvc.AddProtocol(disc.Protocol())

	debugapi.RegisterAPI(srv, topo, p2pSvc, opts.Logger.With("component", "debugapi"))

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

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", opts.RPCPort))
	if err != nil {
		return nil, err
	}
	grpcServer := grpc.NewServer()

	preconfSigner := preconfsigner.NewSigner(opts.PrivKey)

	switch opts.PeerType {
	case p2p.PeerTypeBuilder.String():
		var bidProcessor preconfirmation.BidProcessor = noOpBidProcessor{}
		if opts.BuilderExposeAPI {
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
			noOpBidProcessor{},
			opts.Logger.With("component", "preconfirmation_protocol"),
		)

		searcherAPI := searcherapi.NewService(
			preconfProto,
			opts.Logger.With("component", "searcherapi"),
		)
		searcherapiv1.RegisterSearcherServer(grpcServer, searcherAPI)
	}

	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			opts.Logger.Error("failed to start grpc server", "err", err)
		}
	}()
	closers = append(closers, lis)

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
