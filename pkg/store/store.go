package store

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	preconfpb "github.com/primevprotocol/mev-commit/gen/go/preconfirmation/v1"
)

type Store struct {
	data                          map[string]uint64
	commitmentsByBlockNumber      map[int64][]*EncryptedPreConfirmationWithDecrypted
	commitmentsByCommitmentHash   map[string]*EncryptedPreConfirmationWithDecrypted
	commitmentByBlockNumberMu     sync.RWMutex
	commitmentsByCommitmentHashMu sync.RWMutex
	mu                            sync.RWMutex
}

type EncryptedPreConfirmationWithDecrypted struct {
	*preconfpb.EncryptedPreConfirmation
	*preconfpb.PreConfirmation
}

func NewStore() *Store {
	return &Store{
		data:                        make(map[string]uint64),
		commitmentsByBlockNumber:    make(map[int64][]*EncryptedPreConfirmationWithDecrypted),
		commitmentsByCommitmentHash: make(map[string]*EncryptedPreConfirmationWithDecrypted),
	}
}

func (s *Store) LastBlock() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if value, exists := s.data["last_block"]; exists {
		return value, nil
	}
	return 0, nil
}

func (s *Store) SetLastBlock(blockNum uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data["last_block"] = blockNum
	return nil
}

func (s *Store) addCommitmentByBlockNumber(blockNum int64, commitment *EncryptedPreConfirmationWithDecrypted) {
	s.commitmentByBlockNumberMu.Lock()
	defer s.commitmentByBlockNumberMu.Unlock()

	s.commitmentsByBlockNumber[blockNum] = append(s.commitmentsByBlockNumber[blockNum], commitment)
}

func (s *Store) addCommitmentByHash(hash string, commitment *EncryptedPreConfirmationWithDecrypted) {
	s.commitmentsByCommitmentHashMu.Lock()
	defer s.commitmentsByCommitmentHashMu.Unlock()

	s.commitmentsByCommitmentHash[hash] = commitment
}

func (s *Store) AddCommitment(commitment *EncryptedPreConfirmationWithDecrypted) {
	s.addCommitmentByBlockNumber(commitment.Bid.BlockNumber, commitment)
	s.addCommitmentByHash(common.Bytes2Hex(commitment.Commitment), commitment)
}

func (s *Store) GetCommitmentsByBlockNumber(blockNum int64) ([]*EncryptedPreConfirmationWithDecrypted, error) {
	s.commitmentByBlockNumberMu.RLock()
	defer s.commitmentByBlockNumberMu.RUnlock()

	if commitments, exists := s.commitmentsByBlockNumber[blockNum]; exists {
		return commitments, nil
	}
	return nil, nil
}

func (s *Store) GetCommitmentByHash(hash string) (*EncryptedPreConfirmationWithDecrypted, error) {
	s.commitmentsByCommitmentHashMu.RLock()
	defer s.commitmentsByCommitmentHashMu.RUnlock()

	if commitment, exists := s.commitmentsByCommitmentHash[hash]; exists {
		return commitment, nil
	}
	return nil, nil
}

func (s *Store) DeleteCommitmentByBlockNumber(blockNum int64) error {
	s.commitmentByBlockNumberMu.Lock()
	defer s.commitmentByBlockNumberMu.Unlock()

	for _, v := range s.commitmentsByBlockNumber[blockNum] {
		err := s.deleteCommitmentByHash(common.Bytes2Hex(v.Commitment))
		if err != nil {
			return err
		}
	}
	delete(s.commitmentsByBlockNumber, blockNum)
	return nil
}

func (s *Store) deleteCommitmentByHash(hash string) error {
	s.commitmentsByCommitmentHashMu.Lock()
	defer s.commitmentsByCommitmentHashMu.Unlock()

	delete(s.commitmentsByCommitmentHash, hash)
	return nil
}
