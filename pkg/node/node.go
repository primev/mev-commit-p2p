package node

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bufbuild/protovalidate-go"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	bidderregistry "github.com/primevprotocol/contracts-abi/clients/BidderRegistry"
	blocktracker "github.com/primevprotocol/contracts-abi/clients/BlockTracker"
	preconf "github.com/primevprotocol/contracts-abi/clients/PreConfCommitmentStore"
	bidderapiv1 "github.com/primevprotocol/mev-commit/gen/go/bidderapi/v1"
	preconfpb "github.com/primevprotocol/mev-commit/gen/go/preconfirmation/v1"
	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/providerapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/allowancemanager"
	"github.com/primevprotocol/mev-commit/pkg/apiserver"
	bidder_registrycontract "github.com/primevprotocol/mev-commit/pkg/contracts/bidder_registry"
	blocktrackercontract "github.com/primevprotocol/mev-commit/pkg/contracts/block_tracker"
	preconfcontract "github.com/primevprotocol/mev-commit/pkg/contracts/preconf"
	provider_registrycontract "github.com/primevprotocol/mev-commit/pkg/contracts/provider_registry"
	"github.com/primevprotocol/mev-commit/pkg/debugapi"
	"github.com/primevprotocol/mev-commit/pkg/discovery"
	"github.com/primevprotocol/mev-commit/pkg/events"
	"github.com/primevprotocol/mev-commit/pkg/evmclient"
	"github.com/primevprotocol/mev-commit/pkg/keyexchange"
	"github.com/primevprotocol/mev-commit/pkg/keykeeper"
	"github.com/primevprotocol/mev-commit/pkg/keykeeper/keysigner"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p"
	"github.com/primevprotocol/mev-commit/pkg/preconfirmation"
	bidderapi "github.com/primevprotocol/mev-commit/pkg/rpc/bidder"
	providerapi "github.com/primevprotocol/mev-commit/pkg/rpc/provider"
	"github.com/primevprotocol/mev-commit/pkg/signer"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfencryptor"
	"github.com/primevprotocol/mev-commit/pkg/store"
	"github.com/primevprotocol/mev-commit/pkg/topology"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	grpcServerDialTimeout = 5 * time.Second
)

type Options struct {
	Version                  string
	KeySigner                keysigner.KeySigner
	Secret                   string
	PeerType                 string
	Logger                   *slog.Logger
	P2PPort                  int
	P2PAddr                  string
	HTTPAddr                 string
	RPCAddr                  string
	Bootnodes                []string
	PreconfContract          string
	BlockTrackerContract     string
	ProviderRegistryContract string
	BidderRegistryContract   string
	RPCEndpoint              string
	WSRPCEndpoint            string
	NatAddr                  string
	TLSCertificateFile       string
	TLSPrivateKeyFile        string
}

type Node struct {
	waitClose func()
	closers   []io.Closer
}

func NewNode(opts *Options) (*Node, error) {
	nd := &Node{
		closers: make([]io.Closer, 0),
	}

	srv := apiserver.New(opts.Version, opts.Logger.With("component", "apiserver"))
	peerType := p2p.FromString(opts.PeerType)

	contractRPC, err := ethclient.Dial(opts.RPCEndpoint)
	if err != nil {
		opts.Logger.Error("failed to connect to rpc", "error", err)
		return nil, err
	}
	evmClient, err := evmclient.New(
		opts.KeySigner,
		evmclient.WrapEthClient(contractRPC),
		opts.Logger.With("component", "evmclient"),
	)
	if err != nil {
		opts.Logger.Error("failed to create evm client", "error", err)
		return nil, err
	}
	nd.closers = append(nd.closers, evmClient)
	srv.MetricsRegistry().MustRegister(evmClient.Metrics()...)

	wsRPC, err := ethclient.Dial(opts.WSRPCEndpoint)
	if err != nil {
		opts.Logger.Error("failed to connect to ws rpc", "error", err)
		return nil, err
	}
	wsEvmClient, err := evmclient.New(
		opts.KeySigner,
		evmclient.WrapEthClient(wsRPC),
		opts.Logger.With("component", "wsevmclient"),
	)
	if err != nil {
		opts.Logger.Error("failed to create ws evm client", "error", err)
		return nil, err
	}
	nd.closers = append(nd.closers, wsEvmClient)

	bidderRegistryContractAddr := common.HexToAddress(opts.BidderRegistryContract)

	bidderRegistry := bidder_registrycontract.New(
		opts.KeySigner.GetAddress(),
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

	var keyKeeper keykeeper.KeyKeeper
	switch opts.PeerType {
	case p2p.PeerTypeProvider.String():
		keyKeeper, err = keykeeper.NewProviderKeyKeeper(opts.KeySigner)
		if err != nil {
			opts.Logger.Error("failed to create provider key keeper", "error", err)
			return nil, errors.Join(err, nd.Close())
		}
	case p2p.PeerTypeBidder.String():
		keyKeeper, err = keykeeper.NewBidderKeyKeeper(opts.KeySigner)
		if err != nil {
			opts.Logger.Error("failed to create bidder key keeper", "error", err)
			return nil, errors.Join(err, nd.Close())
		}
	default:
		keyKeeper = keykeeper.NewBaseKeyKeeper(opts.KeySigner)
	}
	p2pSvc, err := libp2p.New(&libp2p.Options{
		KeyKeeper:      keyKeeper,
		Secret:         opts.Secret,
		PeerType:       peerType,
		Register:       providerRegistry,
		Logger:         opts.Logger.With("component", "p2p"),
		ListenPort:     opts.P2PPort,
		ListenAddr:     opts.P2PAddr,
		MetricsReg:     srv.MetricsRegistry(),
		BootstrapAddrs: opts.Bootnodes,
		NatAddr:        opts.NatAddr,
	})
	if err != nil {
		opts.Logger.Error("failed to create p2p service", "error", err)
		return nil, err
	}
	nd.closers = append(nd.closers, p2pSvc)

	topo := topology.New(p2pSvc, opts.Logger.With("component", "topology"))
	disc := discovery.New(topo, p2pSvc, opts.Logger.With("component", "discovery_protocol"))
	nd.closers = append(nd.closers, disc)

	srv.RegisterMetricsCollectors(topo.Metrics()...)

	// Set the announcer for the topology service
	topo.SetAnnouncer(disc)
	// Set the notifier for the p2p service
	p2pSvc.SetNotifier(topo)

	// Register the discovery protocol with the p2p service
	p2pSvc.AddStreamHandlers(disc.Streams()...)

	debugapi.RegisterAPI(srv, topo, p2pSvc, opts.Logger.With("component", "debugapi"))

	ctx, cancel := context.WithCancel(context.Background())

	var preconfProtoClosed <-chan struct{}
	st := store.NewStore()

	contracts, err := getContractABIs(opts)
	if err != nil {
		opts.Logger.Error("failed to get contract ABIs", "error", err)
		cancel()
		return nil, err
	}

	evtMgr := events.NewListener(
		opts.Logger.With("component", "events"),
		evmClient,
		st,
		contracts,
	)

	evtMgrDone := evtMgr.Start(ctx)

	if opts.PeerType != p2p.PeerTypeBootnode.String() {
		lis, err := net.Listen("tcp", opts.RPCAddr)
		if err != nil {
			opts.Logger.Error("failed to listen", "error", err)
			cancel()
			return nil, errors.Join(err, nd.Close())
		}

		var tlsCredentials credentials.TransportCredentials
		if opts.TLSCertificateFile != "" && opts.TLSPrivateKeyFile != "" {
			tlsCredentials, err = credentials.NewServerTLSFromFile(
				opts.TLSCertificateFile,
				opts.TLSPrivateKeyFile,
			)
			if err != nil {
				opts.Logger.Error("failed to load TLS credentials", "error", err)
				cancel()
				return nil, fmt.Errorf("unable to load TLS credentials: %w", err)
			}
		}

		grpcServer := grpc.NewServer(grpc.Creds(tlsCredentials))
		preconfEncryptor := preconfencryptor.NewEncryptor(keyKeeper)
		validator, err := protovalidate.New()
		if err != nil {
			opts.Logger.Error("failed to create proto validator", "error", err)
			cancel()
			return nil, errors.Join(err, nd.Close())
		}

		var (
			bidProcessor preconfirmation.BidProcessor     = noOpBidProcessor{}
			allowanceMgr preconfirmation.AllowanceManager = noOpAllowanceManager{}
		)

		blockTrackerAddr := common.HexToAddress(opts.BlockTrackerContract)

		blockTracker := blocktrackercontract.New(
			blockTrackerAddr,
			evmClient,
			wsEvmClient,
			opts.Logger.With("component", "blocktrackercontract"),
		)

		preconfContractAddr := common.HexToAddress(opts.PreconfContract)
		commitmentDA := preconfcontract.New(
			preconfContractAddr,
			evmClient,
			opts.Logger.With("component", "preconfcontract"),
		)
		opts.Logger.Info("registered preconf contract")

		store := store.NewStore()

		switch opts.PeerType {
		case p2p.PeerTypeProvider.String():
			providerAPI := providerapi.NewService(
				opts.Logger.With("component", "providerapi"),
				providerRegistry,
				opts.KeySigner.GetAddress(),
				evmClient,
				validator,
			)
			providerapiv1.RegisterProviderServer(grpcServer, providerAPI)
			bidProcessor = providerAPI
			srv.RegisterMetricsCollectors(providerAPI.Metrics()...)
			allowanceMgr = allowancemanager.NewAllowanceManager(bidderRegistry,
				blockTracker,
				commitmentDA,
				store,
				evtMgr,
				opts.Logger.With("component", "allowancemanager"),
			)
			allowanceMgr.Start(ctx)
			preconfProto := preconfirmation.New(
				keyKeeper.GetAddress(),
				topo,
				p2pSvc,
				preconfEncryptor,
				allowanceMgr,
				bidProcessor,
				commitmentDA,
				blockTracker,
				evtMgr,
				store,
				opts.Logger.With("component", "preconfirmation_protocol"),
			)

			preconfProtoClosed = preconfProto.Start(ctx)

			// Only register handler for provider
			p2pSvc.AddStreamHandlers(preconfProto.Streams()...)
			keyexchange := keyexchange.New(
				topo,
				p2pSvc,
				keyKeeper,
				opts.Logger.With("component", "keyexchange_protocol"),
				signer.New(),
			)
			p2pSvc.AddStreamHandlers(keyexchange.Streams()...)
			srv.RegisterMetricsCollectors(preconfProto.Metrics()...)

		case p2p.PeerTypeBidder.String():
			preconfProto := preconfirmation.New(
				keyKeeper.GetAddress(),
				topo,
				p2pSvc,
				preconfEncryptor,
				allowanceMgr,
				bidProcessor,
				commitmentDA,
				blockTracker,
				evtMgr,
				store,
				opts.Logger.With("component", "preconfirmation_protocol"),
			)

			preconfProtoClosed = preconfProto.Start(ctx)

			srv.RegisterMetricsCollectors(preconfProto.Metrics()...)

			bidderAPI := bidderapi.NewService(
				preconfProto,
				opts.KeySigner.GetAddress(),
				bidderRegistry,
				blockTracker,
				validator,
				opts.Logger.With("component", "bidderapi"),
			)
			bidderapiv1.RegisterBidderServer(grpcServer, bidderAPI)

			keyexchange := keyexchange.New(
				topo,
				p2pSvc,
				keyKeeper,
				opts.Logger.With("component", "keyexchange_protocol"),
				signer.New(),
			)
			topo.SubscribePeer(func(p p2p.Peer) {
				if p.Type == p2p.PeerTypeProvider {
					err = keyexchange.SendTimestampMessage()
					if err != nil {
						opts.Logger.Error("failed to send timestamp message", "error", err)
					}
				}
			})

			srv.RegisterMetricsCollectors(bidderAPI.Metrics()...)
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

		// Since we don't know if the server has TLS enabled on its rpc
		// endpoint, we try different strategies from most secure to
		// least secure. In the future, when only TLS-enabled servers
		// are allowed, only the TLS system pool certificate strategy
		// should be used.
		var grpcConn *grpc.ClientConn
		for _, e := range []struct {
			strategy   string
			isSecure   bool
			credential credentials.TransportCredentials
		}{
			{"TLS system pool certificate", true, credentials.NewClientTLSFromCert(nil, "")},
			{"TLS skip verification", false, credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})},
			{"TLS disabled", false, insecure.NewCredentials()},
		} {
			ctx, cancel := context.WithTimeout(context.Background(), grpcServerDialTimeout)
			opts.Logger.Info("dialing to grpc server", "strategy", e.strategy)
			grpcConn, err = grpc.DialContext(
				ctx,
				opts.RPCAddr,
				grpc.WithBlock(),
				grpc.WithTransportCredentials(e.credential),
			)
			if err != nil {
				opts.Logger.Error("failed to dial grpc server", "error", err)
				cancel()
				continue
			}

			cancel()
			if !e.isSecure {
				opts.Logger.Warn("established connection with the grpc server has potential security risk")
			}
			break
		}
		if grpcConn == nil {
			cancel()
			return nil, errors.New("dialing of grpc server failed")
		}

		handlerCtx, handlerCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer handlerCancel()

		gatewayMux := runtime.NewServeMux()
		switch opts.PeerType {
		case p2p.PeerTypeProvider.String():
			err := providerapiv1.RegisterProviderHandler(handlerCtx, gatewayMux, grpcConn)
			if err != nil {
				opts.Logger.Error("failed to register provider handler", "err", err)
				cancel()
				return nil, errors.Join(err, nd.Close())
			}
		case p2p.PeerTypeBidder.String():
			err := bidderapiv1.RegisterBidderHandler(handlerCtx, gatewayMux, grpcConn)
			if err != nil {
				opts.Logger.Error("failed to register bidder handler", "err", err)
				cancel()
				return nil, errors.Join(err, nd.Close())
			}
		}

		srv.ChainHandlers("/", gatewayMux)
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
		opts.Logger.Info("grpc server connected and handlers are started", "state", grpcConn.GetState())
	}

	server := &http.Server{
		Addr:    opts.HTTPAddr,
		Handler: srv.Router(),
	}

	go func() {
		var (
			err        error
			tlsEnabled = opts.TLSCertificateFile != "" && opts.TLSPrivateKeyFile != ""
		)
		opts.Logger.Info("starting to listen", "tls", tlsEnabled)
		if tlsEnabled {
			err = server.ListenAndServeTLS(
				opts.TLSCertificateFile,
				opts.TLSPrivateKeyFile,
			)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			opts.Logger.Error("failed to start server", "err", err)
		}
	}()
	nd.closers = append(nd.closers, server)

	nd.waitClose = func() {
		cancel()

		closeChan := make(chan struct{})
		go func() {
			defer close(closeChan)

			<-evtMgrDone
			<-preconfProtoClosed
		}()

		<-closeChan
	}

	return nd, nil
}

func getContractABIs(opts *Options) (map[common.Address]*abi.ABI, error) {
	abis := make(map[common.Address]*abi.ABI)

	btABI, err := abi.JSON(strings.NewReader(blocktracker.BlocktrackerABI))
	if err != nil {
		return nil, err
	}
	abis[common.HexToAddress(opts.BlockTrackerContract)] = &btABI

	pcABI, err := abi.JSON(strings.NewReader(preconf.PreconfcommitmentstoreABI))
	if err != nil {
		return nil, err
	}
	abis[common.HexToAddress(opts.PreconfContract)] = &pcABI

	brABI, err := abi.JSON(strings.NewReader(bidderregistry.BidderregistryABI))
	if err != nil {
		return nil, err
	}
	abis[common.HexToAddress(opts.BidderRegistryContract)] = &brABI

	return abis, nil
}

func (n *Node) Close() error {
	workersClosed := make(chan struct{})
	go func() {
		defer close(workersClosed)

		if n.waitClose != nil {
			n.waitClose()
		}
	}()

	var err error
	for _, c := range n.closers {
		err = errors.Join(err, c.Close())
	}

	return err
}

type noOpBidProcessor struct{}

// ProcessBid auto accepts all bids sent.
func (noOpBidProcessor) ProcessBid(
	_ context.Context,
	_ *preconfpb.Bid,
) (chan providerapiv1.BidResponse_Status, error) {
	statusC := make(chan providerapiv1.BidResponse_Status, 5)
	statusC <- providerapiv1.BidResponse_STATUS_ACCEPTED
	close(statusC)

	return statusC, nil
}

type noOpAllowanceManager struct{}

func (noOpAllowanceManager) Start(_ context.Context) <-chan struct{} {
	return nil
}

func (noOpAllowanceManager) CheckAllowance(_ context.Context, _ common.Address) error {
	return nil
}
