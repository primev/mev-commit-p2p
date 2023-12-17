package node

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	bidderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/bidderapi/v1"
	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/providerapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/apiserver"
	bidder_registrycontract "github.com/primevprotocol/mev-commit/pkg/contracts/bidder_registry"
	preconfcontract "github.com/primevprotocol/mev-commit/pkg/contracts/preconf"
	provider_registrycontract "github.com/primevprotocol/mev-commit/pkg/contracts/provider_registry"
	"github.com/primevprotocol/mev-commit/pkg/debugapi"
	"github.com/primevprotocol/mev-commit/pkg/discovery"
	"github.com/primevprotocol/mev-commit/pkg/evmclient"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p"
	"github.com/primevprotocol/mev-commit/pkg/preconfirmation"
	"github.com/primevprotocol/mev-commit/pkg/register"
	bidderapi "github.com/primevprotocol/mev-commit/pkg/rpc/bidder"
	providerapi "github.com/primevprotocol/mev-commit/pkg/rpc/provider"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
	"github.com/primevprotocol/mev-commit/pkg/topology"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type Options struct {
	Version                  string
	PrivKey                  *ecdsa.PrivateKey
	Secret                   string
	PeerType                 string
	Logger                   *slog.Logger
	P2PPort                  int
	HTTPPort                 int
	RPCPort                  int
	Bootnodes                []string
	ExposeProviderAPI        bool
	PreconfContract          string
	ProviderRegistryContract string
	BidderRegistryContract   string
	RPCEndpoint              string
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
	peerType := p2p.FromString(opts.PeerType)
	ownerEthAddress := libp2p.GetEthAddressFromPubKey(&opts.PrivKey.PublicKey)

	contractRPC, err := ethclient.Dial(opts.RPCEndpoint)
	if err != nil {
		return nil, err
	}
	evmClient, err := evmclient.New(
		ownerEthAddress,
		opts.PrivKey,
		evmclient.WrapEthClient(contractRPC),
		opts.Logger.With("component", "evmclient"),
	)
	if err != nil {
		return nil, err
	}
	nd.closers = append(nd.closers, evmClient)

	srv.MetricsRegistry().MustRegister(evmClient.Metrics()...)

	bidderRegistryContractAddr := common.HexToAddress(opts.BidderRegistryContract)

	bidderRegistry := bidder_registrycontract.New(
		bidderRegistryContractAddr,
		evmClient,
		opts.Logger.With("component", "bidderregistry"),
	)

	providerRegistryContractAddr := common.HexToAddress(opts.ProviderRegistryContract)

	providerRegistry := provider_registrycontract.New(
		providerRegistryContractAddr,
		evmClient,
		opts.Logger.With("component", "providerregistry"),
	)

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

		var (
			bidProcessor preconfirmation.BidProcessor = noOpBidProcessor{}
			commitmentDA preconfcontract.Interface    = noOpCommitmentDA{}
		)

		switch opts.PeerType {
		case p2p.PeerTypeProvider.String():
			if opts.ExposeProviderAPI {
				providerAPI := providerapi.NewService(
					opts.Logger.With("component", "providerapi"),
					providerRegistry,
					ownerEthAddress,
					evmClient,
				)
				providerapiv1.RegisterProviderServer(grpcServer, providerAPI)
				bidProcessor = providerAPI
			}

			preconfContractAddr := common.HexToAddress(opts.PreconfContract)

			commitmentDA = preconfcontract.New(
				preconfContractAddr,
				evmClient,
				opts.Logger.With("component", "preconfcontract"),
			)

			preconfProto := preconfirmation.New(
				topo,
				p2pSvc,
				preconfSigner,
				bidderRegistry,
				bidProcessor,
				commitmentDA,
				opts.Logger.With("component", "preconfirmation_protocol"),
			)
			// Only register handler for provider
			p2pSvc.AddProtocol(preconfProto.Protocol())

		case p2p.PeerTypeBidder.String():
			preconfProto := preconfirmation.New(
				topo,
				p2pSvc,
				preconfSigner,
				bidderRegistry,
				bidProcessor,
				commitmentDA,
				opts.Logger.With("component", "preconfirmation_protocol"),
			)

			bidderAPI := bidderapi.NewService(
				preconfProto,
				ownerEthAddress,
				bidderRegistry,
				opts.Logger.With("component", "bidderapi"),
			)
			bidderapiv1.RegisterBidderServer(grpcServer, bidderAPI)
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
		case p2p.PeerTypeProvider.String():
			err := providerapiv1.RegisterProviderHandler(bgCtx, gwMux, grpcConn)
			if err != nil {
				opts.Logger.Error("failed to register provider handler", "err", err)
				return nil, err
			}
		case p2p.PeerTypeBidder.String():
			err := bidderapiv1.RegisterBidderHandler(bgCtx, gwMux, grpcConn)
			if err != nil {
				opts.Logger.Error("failed to register bidder handler", "err", err)
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
	nd.closers = append(nd.closers, server)

	return nd, nil
}

func (n *Node) Close() error {
	var err error
	for _, c := range n.closers {
		err = errors.Join(err, c.Close())
	}

	return err
}

type noOpBidProcessor struct{}

// The noOpBidProcesor auto accepts all bids sent
func (noOpBidProcessor) ProcessBid(
	_ context.Context,
	_ *preconfsigner.Bid,
) (chan providerapiv1.BidResponse_Status, error) {
	statusC := make(chan providerapiv1.BidResponse_Status, 5)
	statusC <- providerapiv1.BidResponse_STATUS_ACCEPTED
	close(statusC)

	return statusC, nil
}

type noOpCommitmentDA struct{}

func (noOpCommitmentDA) StoreCommitment(
	_ context.Context,
	_ *big.Int,
	_ uint64,
	_ string,
	_ []byte,
	_ []byte,
) error {
	return nil
}

func (noOpCommitmentDA) Close() error {
	return nil
}
