package evmclient

import (
	"context"
	"errors"
	"log/slog"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	maxSentTxs uint64 = 256
)

var (
	ErrTxnCancelled  = errors.New("transaction was cancelled")
	ErrMonitorClosed = errors.New("monitor was closed")
)

type txmonitor struct {
	baseCtx            context.Context
	baseCancel         context.CancelFunc
	owner              common.Address
	mtx                sync.Mutex
	waitMap            map[uint64]map[common.Hash][]chan Result
	client             EVM
	newTxAdded         chan struct{}
	waitDone           chan struct{}
	logger             *slog.Logger
	metrics            *metrics
	lastConfirmedNonce atomic.Uint64
}

func newTxMonitor(
	owner common.Address,
	client EVM,
	logger *slog.Logger,
	m *metrics,
) *txmonitor {
	baseCtx, baseCancel := context.WithCancel(context.Background())
	if m == nil {
		m = newMetrics()
	}
	tm := &txmonitor{
		baseCtx:    baseCtx,
		baseCancel: baseCancel,
		owner:      owner,
		client:     client,
		logger:     logger,
		metrics:    m,
		waitMap:    make(map[uint64]map[common.Hash][]chan Result),
		newTxAdded: make(chan struct{}),
		waitDone:   make(chan struct{}),
	}
	go tm.watchLoop()
	return tm
}

type Result struct {
	Receipt *types.Receipt
	Err     error
}

func (t *txmonitor) watchLoop() {
	defer close(t.waitDone)

	queryTicker := time.NewTicker(500 * time.Millisecond)
	defer queryTicker.Stop()

	defer func() {
		t.mtx.Lock()
		defer t.mtx.Unlock()

		for _, v := range t.waitMap {
			for _, c := range v {
				for _, c := range c {
					c <- Result{nil, ErrMonitorClosed}
					close(c)
				}
			}
		}
	}()

	lastBlock := uint64(0)
	for {
		newTx := false
		select {
		case <-t.baseCtx.Done():
			return
		case <-t.newTxAdded:
			newTx = true
		case <-queryTicker.C:
		}

		if len(t.waitMap) == 0 {
			continue
		}

		currentBlock, err := t.client.BlockNumber(t.baseCtx)
		if err != nil {
			t.logger.Error("failed to get block number", "err", err)
			continue
		}

		if currentBlock <= lastBlock && !newTx {
			continue
		}

		t.check(currentBlock)
		lastBlock = currentBlock
	}
}

func (t *txmonitor) Close() error {
	t.baseCancel()
	select {
	case <-t.waitDone:
		return nil
	case <-time.After(10 * time.Second):
		return errors.New("failed to close txmonitor")
	}
}

func (t *txmonitor) getOlderTxns(nonce uint64) map[uint64][]common.Hash {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	txnMap := make(map[uint64][]common.Hash)
	for k, v := range t.waitMap {
		if k >= nonce {
			continue
		}

		for h := range v {
			txnMap[k] = append(txnMap[k], h)
		}
	}

	return txnMap
}

func (t *txmonitor) notify(
	nonce uint64,
	txn common.Hash,
	res Result,
) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	for _, c := range t.waitMap[nonce][txn] {
		c <- res
		close(c)
	}
	delete(t.waitMap[nonce], txn)
}

func (t *txmonitor) check(newBlock uint64) {
	lastNonce, err := t.client.NonceAt(t.baseCtx, t.owner, new(big.Int).SetUint64(newBlock))
	if err != nil {
		t.logger.Error("failed to get nonce", "err", err)
		return
	}

	t.lastConfirmedNonce.Store(lastNonce)
	t.metrics.LastConfirmedNonce.Set(float64(lastNonce))

	checkTxns := t.getOlderTxns(lastNonce)

	for nonce, txns := range checkTxns {
		for _, txn := range txns {
			receipt, err := t.client.TransactionReceipt(t.baseCtx, txn)
			if err != nil {
				// If in future we move away from PoA, then we should check for
				// check for reorgs of the blockchain here. Even though the txn
				// is not found, it can still become part of the chain because of reorgs.
				if errors.Is(err, ethereum.NotFound) {
					t.notify(nonce, txn, Result{nil, ErrTxnCancelled})
					continue
				}
				t.logger.Error("failed to get receipt", "err", err)
				return
			}

			if receipt == nil {
				continue
			}

			t.notify(nonce, txn, Result{receipt, nil})
		}
	}
}

func (t *txmonitor) allowNonce(nonce uint64) bool {
	return nonce <= t.lastConfirmedNonce.Load()+maxSentTxs
}

func (t *txmonitor) watchTx(txHash common.Hash, nonce uint64) (<-chan Result, error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.waitMap[nonce] == nil {
		t.waitMap[nonce] = make(map[common.Hash][]chan Result)
	}

	c := make(chan Result, 1)
	t.waitMap[nonce][txHash] = append(t.waitMap[nonce][txHash], c)

	select {
	case t.newTxAdded <- struct{}{}:
	default:
	}
	return c, nil
}
