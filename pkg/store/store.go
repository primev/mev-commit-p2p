package store

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	preconfpb "github.com/primevprotocol/mev-commit/gen/go/preconfirmation/v1"
)

type Store struct {
	*BlockStore
	*CommitmentsStore
	*BidderBalancesStore
}

type BlockStore struct {
	data map[string]uint64
	mu   sync.RWMutex
}

type CommitmentsStore struct {
	commitmentsByBlockNumber      map[int64][]*EncryptedPreConfirmationWithDecrypted
	commitmentsByCommitmentHash   map[string]*EncryptedPreConfirmationWithDecrypted
	commitmentByBlockNumberMu     sync.RWMutex
	commitmentsByCommitmentHashMu sync.RWMutex
}

type EncryptedPreConfirmationWithDecrypted struct {
	*preconfpb.EncryptedPreConfirmation
	*preconfpb.PreConfirmation
}

func NewStore() *Store {
	return &Store{
		BlockStore: &BlockStore{
			data: make(map[string]uint64),
		},
		CommitmentsStore: &CommitmentsStore{
			commitmentsByBlockNumber:    make(map[int64][]*EncryptedPreConfirmationWithDecrypted),
			commitmentsByCommitmentHash: make(map[string]*EncryptedPreConfirmationWithDecrypted),
		},
		BidderBalancesStore: &BidderBalancesStore{
			balances: make(map[string]*big.Int),
		},
	}
}

func (bs *BlockStore) LastBlock() (uint64, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if value, exists := bs.data["last_block"]; exists {
		return value, nil
	}
	return 0, nil
}

func (bs *BlockStore) SetLastBlock(blockNum uint64) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.data["last_block"] = blockNum
	return nil
}

func (cs *CommitmentsStore) addCommitmentByBlockNumber(blockNum int64, commitment *EncryptedPreConfirmationWithDecrypted) {
	cs.commitmentByBlockNumberMu.Lock()
	defer cs.commitmentByBlockNumberMu.Unlock()

	cs.commitmentsByBlockNumber[blockNum] = append(cs.commitmentsByBlockNumber[blockNum], commitment)
}

func (cs *CommitmentsStore) addCommitmentByHash(hash string, commitment *EncryptedPreConfirmationWithDecrypted) {
	cs.commitmentsByCommitmentHashMu.Lock()
	defer cs.commitmentsByCommitmentHashMu.Unlock()

	cs.commitmentsByCommitmentHash[hash] = commitment
}

func (cs *CommitmentsStore) AddCommitment(commitment *EncryptedPreConfirmationWithDecrypted) {
	cs.addCommitmentByBlockNumber(commitment.Bid.BlockNumber, commitment)
	cs.addCommitmentByHash(common.Bytes2Hex(commitment.Commitment), commitment)
}

func (cs *CommitmentsStore) GetCommitmentsByBlockNumber(blockNum int64) ([]*EncryptedPreConfirmationWithDecrypted, error) {
	cs.commitmentByBlockNumberMu.RLock()
	defer cs.commitmentByBlockNumberMu.RUnlock()

	if commitments, exists := cs.commitmentsByBlockNumber[blockNum]; exists {
		return commitments, nil
	}
	return nil, nil
}

func (cs *CommitmentsStore) GetCommitmentByHash(hash string) (*EncryptedPreConfirmationWithDecrypted, error) {
	cs.commitmentsByCommitmentHashMu.RLock()
	defer cs.commitmentsByCommitmentHashMu.RUnlock()

	if commitment, exists := cs.commitmentsByCommitmentHash[hash]; exists {
		return commitment, nil
	}
	return nil, nil
}

func (cs *CommitmentsStore) DeleteCommitmentByBlockNumber(blockNum int64) error {
	cs.commitmentByBlockNumberMu.Lock()
	defer cs.commitmentByBlockNumberMu.Unlock()

	for _, v := range cs.commitmentsByBlockNumber[blockNum] {
		err := cs.deleteCommitmentByHash(common.Bytes2Hex(v.Commitment))
		if err != nil {
			return err
		}
	}
	delete(cs.commitmentsByBlockNumber, blockNum)
	return nil
}

func (cs *CommitmentsStore) deleteCommitmentByHash(hash string) error {
	cs.commitmentsByCommitmentHashMu.Lock()
	defer cs.commitmentsByCommitmentHashMu.Unlock()

	delete(cs.commitmentsByCommitmentHash, hash)
	return nil
}

type BidderBalancesStore struct {
	balances map[string]*big.Int
	mu       sync.RWMutex
}

func (bbs *BidderBalancesStore) SetBalance(bidder common.Address, windowNumber *big.Int, prepaidAmount *big.Int) error {
	bbs.mu.Lock()
	defer bbs.mu.Unlock()
	bssKey := getBBSKey(bidder, windowNumber)
	bbs.balances[bssKey] = prepaidAmount
	return nil
}

func (bbs *BidderBalancesStore) GetBalance(bidder common.Address, windowNumber *big.Int) (*big.Int, error) {
	bbs.mu.RLock()
	defer bbs.mu.RUnlock()
	bssKey := getBBSKey(bidder, windowNumber)
	if balance, exists := bbs.balances[bssKey]; exists {
		return balance, nil
	}
	return nil, nil
}

func getBBSKey(bidder common.Address, windowNumber *big.Int) string {
	return bidder.String() + windowNumber.String()
}