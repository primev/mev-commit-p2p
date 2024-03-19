package blobinclusion

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"slices"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

const (
	ProtocolName    = "blobinclusion"
	ProtocolVersion = "1.0.0"
)

type BlobInclusion struct {
	self           p2p.Peer
	topo           Topology
	streamer       p2p.Streamer
	logger         *slog.Logger
	inclusionLists *lru.Cache[int64, InclusionLists]
	inclusionMu    sync.Mutex
}

type Topology interface {
	GetPeers(topology.Query) []p2p.Peer
	SubscribeOnConnected(func(p2p.Peer))
}

type InclusionLists struct {
	RelayBlobs map[string][]BlobInclusionItem
}

func New(
	self p2p.Peer,
	topo Topology,
	streamer p2p.Streamer,
	logger *slog.Logger,
) *BlobInclusion {
	cache, err := lru.New[int64, InclusionLists](1000) // 1000 blocks
	if err != nil {
		// out of memory
		panic(err)
	}
	p := &BlobInclusion{
		self:           self,
		topo:           topo,
		streamer:       streamer,
		logger:         logger,
		inclusionLists: cache,
	}
	topo.SubscribeOnConnected(func(peer p2p.Peer) {
		if peer.Type == p2p.PeerTypeProvider {
			go func() {
				p.SendBlobs(context.Background(), peer)
			}()
		}
	})
	return p
}

func (p *BlobInclusion) Protocol() p2p.ProtocolSpec {
	return p2p.ProtocolSpec{
		Name:    ProtocolName,
		Version: ProtocolVersion,
		StreamSpecs: []p2p.StreamSpec{
			{
				Name:    "blob",
				Handler: p.handleBlob,
			},
		},
	}
}

type BlobInclusionItem struct {
	TxHash      string
	BlockNumber *big.Int
}

type Ack struct {
	Status bool
}

func (p *BlobInclusion) sendBlobToProvider(
	ctx context.Context,
	provider p2p.Peer,
	blobs ...BlobInclusionItem,
) (*Ack, error) {

	providerStream, err := p.streamer.NewStream(
		ctx,
		provider,
		ProtocolName,
		ProtocolVersion,
		"blob",
	)
	if err != nil {
		return nil, fmt.Errorf("creating stream: %w", err)
	}

	p.logger.Info("sending blob", "blobs", len(blobs), "provider", provider)

	r, w := msgpack.NewReaderWriter[Ack, BlobInclusionItem](providerStream)
	for _, b := range blobs {
		err = w.WriteMsg(ctx, &b)
		if err != nil {
			_ = providerStream.Reset()
			return nil, fmt.Errorf("writing message: %w", err)
		}
	}

	_ = providerStream.CloseWrite()

	ack, err := r.ReadMsg(ctx)
	if err != nil {
		_ = providerStream.Reset()
		return nil, fmt.Errorf("reading message: %w", err)
	}

	_ = providerStream.Close()
	p.logger.Info("received blob ack", "provider", provider, "ack", ack)

	return ack, nil
}

func (p *BlobInclusion) addItem(
	blockNumber int64,
	blob BlobInclusionItem,
) {
	p.inclusionMu.Lock()
	defer p.inclusionMu.Unlock()

	currentList, ok := p.inclusionLists.Get(blockNumber)
	if !ok {
		currentList = InclusionLists{
			RelayBlobs: make(map[string][]BlobInclusionItem),
		}
	}
	if currentBlobs, ok := currentList.RelayBlobs[p.self.EthAddress.Hex()]; ok {
		if !slices.Contains(currentBlobs, blob) {
			currentBlobs = append(currentBlobs, blob)
			currentList.RelayBlobs[p.self.EthAddress.Hex()] = currentBlobs
		}
	} else {
		currentList.RelayBlobs[p.self.EthAddress.Hex()] = []BlobInclusionItem{blob}
	}
	p.inclusionLists.Add(blockNumber, currentList)
}

func (p *BlobInclusion) SendBlobs(
	ctx context.Context,
	provider p2p.Peer,
) {
	for _, blockNumber := range p.inclusionLists.Keys() {
		list, ok := p.inclusionLists.Get(blockNumber)
		if !ok {
			continue
		}
		blobList, ok := list.RelayBlobs[p.self.EthAddress.Hex()]
		if !ok {
			continue
		}
		_, err := p.sendBlobToProvider(ctx, provider, blobList...)
		if err != nil {
			p.logger.Error("sending blobs", "error", err)
		}
	}
}

func (p *BlobInclusion) BroadcastBlob(
	ctx context.Context,
	txHash string,
	blockNumber *big.Int,
) ([]p2p.Peer, error) {
	providers := p.topo.GetPeers(topology.Query{Type: p2p.PeerTypeProvider})
	if len(providers) == 0 {
		p.logger.Error("no providers available", "txHash", txHash)
		return nil, errors.New("no providers available")
	}

	blobHashes := strings.Split(txHash, ",")
	blobs := make([]BlobInclusionItem, 0, len(blobHashes))
	for _, hash := range blobHashes {
		b := BlobInclusionItem{
			TxHash:      hash,
			BlockNumber: blockNumber,
		}
		blobs = append(blobs, b)
		p.addItem(blockNumber.Int64(), b)
	}

	type peerAck struct {
		peer p2p.Peer
		ack  *Ack
	}

	ackC := make(chan peerAck, len(providers))

	wg := sync.WaitGroup{}
	for idx := range providers {
		wg.Add(1)
		go func(provider p2p.Peer) {
			defer wg.Done()

			logger := p.logger.With("provider", provider, "blob", txHash)

			ack, err := p.sendBlobToProvider(ctx, provider, blobs...)
			if err != nil {
				logger.Error("sending blob", "error", err)
				return
			}

			select {
			case ackC <- peerAck{peer: provider, ack: ack}:
			case <-ctx.Done():
				logger.Error("context cancelled", "error", ctx.Err())
				return
			}
		}(providers[idx])
	}

	go func() {
		wg.Wait()
		close(ackC)
	}()

	confirmedProviders := make([]p2p.Peer, 0, len(providers))
	for ack := range ackC {
		if !ack.ack.Status {
			continue
		}
		confirmedProviders = append(confirmedProviders, ack.peer)
	}

	return confirmedProviders, nil
}

var ErrInvalidNodeTypeForBlob = errors.New("invalid sender type for blob")

func (p *BlobInclusion) handleBlob(
	ctx context.Context,
	peer p2p.Peer,
	stream p2p.Stream,
) error {
	if peer.Type != p2p.PeerTypeRelay {
		return ErrInvalidNodeTypeForBlob
	}

	r, w := msgpack.NewReaderWriter[BlobInclusionItem, Ack](stream)

	for {
		blob, err := r.ReadMsg(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		p.logger.Info("received blob", "blob", blob)
		p.addItem(blob.BlockNumber.Int64(), *blob)
	}

	return w.WriteMsg(ctx, &Ack{Status: true})
}

func (p *BlobInclusion) GetInclusionLists(
	blockNumber int64,
) InclusionLists {
	list, ok := p.inclusionLists.Get(blockNumber)
	if !ok {
		return InclusionLists{}
	}
	return list
}
