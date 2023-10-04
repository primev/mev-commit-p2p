package debugapi

import (
	"log/slog"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/apiserver"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

type APIServer interface {
	ChainHandlers(string, http.Handler, ...func(http.Handler) http.Handler)
}

func RegisterAPI(
	srv APIServer,
	topo topology.Topology,
	p2pSvc *libp2p.Service,
	logger *slog.Logger,
) {
	d := &debugapi{
		topo:   topo,
		p2p:    p2pSvc,
		logger: logger,
	}

	srv.ChainHandlers(
		"/topology",
		apiserver.MethodHandler("GET", d.handleTopology),
	)
}

type debugapi struct {
	topo   topology.Topology
	p2p    *libp2p.Service
	logger *slog.Logger
}

type topologyResponse struct {
	Self           map[string]interface{}      `json:"self"`
	ConnectedPeers map[string][]common.Address `json:"connected_peers"`
}

func (d *debugapi) handleTopology(w http.ResponseWriter, r *http.Request) {
	logger := d.logger.With("method", "handleTopology")
	builders := d.topo.GetPeers(topology.Query{Type: p2p.PeerTypeBuilder})
	searchers := d.topo.GetPeers(topology.Query{Type: p2p.PeerTypeSearcher})

	topoResp := topologyResponse{
		Self:           d.p2p.Self(),
		ConnectedPeers: make(map[string][]common.Address),
	}

	if len(builders) > 0 {
		connectedBuilders := make([]common.Address, len(builders))
		for _, builder := range builders {
			connectedBuilders = append(connectedBuilders, builder.EthAddress)
		}
		topoResp.ConnectedPeers["builders"] = connectedBuilders
	}
	if len(searchers) > 0 {
		connectedSearchers := make([]common.Address, len(searchers))
		for _, searcher := range searchers {
			connectedSearchers = append(connectedSearchers, searcher.EthAddress)
		}
		topoResp.ConnectedPeers["searchers"] = connectedSearchers
	}

	err := apiserver.WriteResponse(w, http.StatusOK, topoResp)
	if err != nil {
		logger.Error("error writing response", "err", err)
	}
}
