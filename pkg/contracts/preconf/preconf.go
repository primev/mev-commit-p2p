package preconfcontract

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	preconfcommitmentstore "github.com/primevprotocol/contracts-abi/clients/PreConfCommitmentStore"
	"github.com/primevprotocol/mev-commit/pkg/evmclient"
)

var preconfABI = func() abi.ABI {
	abi, err := abi.JSON(strings.NewReader(preconfcommitmentstore.PreconfcommitmentstoreMetaData.ABI))
	if err != nil {
		panic(err)
	}
	return abi
}

var defaultWaitTimeout = 10 * time.Second

type Interface interface {
	StoreEncryptedCommitment(
		ctx context.Context,
		commitmentDigest []byte,
		commitmentSignature []byte,
	) (common.Hash, error)
	OpenCommitment(
		ctx context.Context,
		encryptedCommitmentIndex []byte,
		bid string,
		blockNumber int64,
		txnHash string,
		decayStartTimeStamp int64,
		decayEndTimeStamp int64,
		bidSignature []byte,
		commitmentSignature []byte,
		sharedSecretKey []byte,
	) (common.Hash, error)
}

type preconfContract struct {
	preconfABI          abi.ABI
	preconfContractAddr common.Address
	client              evmclient.Interface
	logger              *slog.Logger
}

func New(
	preconfContractAddr common.Address,
	client evmclient.Interface,
	logger *slog.Logger,
) Interface {
	return &preconfContract{
		preconfABI:          preconfABI(),
		preconfContractAddr: preconfContractAddr,
		client:              client,
		logger:              logger,
	}
}

func (p *preconfContract) StoreEncryptedCommitment(
	ctx context.Context,
	commitmentDigest []byte,
	commitmentSignature []byte,
) (common.Hash, error) {
	callData, err := p.preconfABI.Pack(
		"storeEncryptedCommitment",
		[32]byte(commitmentDigest),
		commitmentSignature,
	)
	if err != nil {
		p.logger.Error("preconf contract storeEncryptedCommitment pack error", "err", err)
		return common.Hash{}, err
	}

	txnHash, err := p.client.Send(ctx, &evmclient.TxRequest{
		To:       &p.preconfContractAddr,
		CallData: callData,
	})
	if err != nil {
		return common.Hash{}, err
	}

	// todo: add event tracker to add commitment to avoid waiting
	receipt, err := p.client.WaitForReceipt(ctx, txnHash)
	if err != nil {
		return common.Hash{}, err // Updated to return common.Hash{}
	}

	p.logger.Info("preconf contract storeEncryptedCommitment successful", "txnHash", txnHash)
	eventTopicHash := p.preconfABI.Events["EncryptedCommitmentStored"].ID // This is the event signature hash

	for _, log := range receipt.Logs {
		if len(log.Topics) > 0 && log.Topics[0] == eventTopicHash {
			commitmentIndex := log.Topics[1] // Topics[0] is the event signature, Topics[1] should be the first indexed argument
			p.logger.Info("Encrypted commitment stored", "commitmentIndex", commitmentIndex.Hex())

			return commitmentIndex, nil // Return the extracted commitmentIndex
		}
	}

	return common.Hash{}, nil
}

func (p *preconfContract) OpenCommitment(
	ctx context.Context,
	encryptedCommitmentIndex []byte,
	bid string,
	blockNumber int64,
	txnHash string,
	decayStartTimeStamp int64,
	decayEndTimeStamp int64,
	bidSignature []byte,
	commitmentSignature []byte,
	sharedSecretKey []byte,
) (common.Hash, error) {
	bidAmt, _ := new(big.Int).SetString(bid, 10)
	callData, err := p.preconfABI.Pack(
		"openCommitment",
		encryptedCommitmentIndex,
		bidAmt,
		big.NewInt(blockNumber),
		txnHash,
		big.NewInt(decayStartTimeStamp),
		big.NewInt(decayEndTimeStamp),
		bidSignature,
		commitmentSignature,
		sharedSecretKey,
	)
	if err != nil {
		p.logger.Error("Error packing call data for openCommitment", "error", err)
		return common.Hash{}, err
	}

	_, err = p.client.Send(ctx, &evmclient.TxRequest{
		To:       &p.preconfContractAddr,
		CallData: callData,
	})
	if err != nil {
		return common.Hash{}, err
	}

	return common.Hash{}, fmt.Errorf("commitmentIndex not found in transaction receipt")
}
