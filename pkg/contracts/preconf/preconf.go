package preconfcontract

import (
	"context"
	"log/slog"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	contractsabi "github.com/primevprotocol/mev-commit/pkg/abi"
	"github.com/primevprotocol/mev-commit/pkg/evmclient"
)

var preconfABI = func() abi.ABI {
	abi, err := abi.JSON(strings.NewReader(contractsabi.PreConfJson))
	if err != nil {
		panic(err)
	}
	return abi
}

type Interface interface {
	StoreCommitment(
		ctx context.Context,
		bid *big.Int,
		blockNumber uint64,
		txHash string,
		bidSignature []byte,
		commitmentSignature []byte,
	) error
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

func (p *preconfContract) StoreCommitment(
	ctx context.Context,
	bid *big.Int,
	blockNumber uint64,
	txHash string,
	bidSignature []byte,
	commitmentSignature []byte,
) error {

	callData, err := p.preconfABI.Pack(
		"storeCommitment",
		uint64(bid.Int64()),
		blockNumber,
		txHash,
		bidSignature,
		commitmentSignature,
	)
	if err != nil {
		p.logger.Error("preconf contract storeCommitment pack error", "err", err)
		return err
	}

	txnHash, err := p.client.Send(ctx, &evmclient.TxRequest{
		To:       &p.preconfContractAddr,
		CallData: callData,
	})
	if err != nil {
		return err
	}

	receipt, err := p.client.WaitForReceipt(ctx, txnHash)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		p.logger.Error(
			"preconf contract storeCommitment receipt error",
			"txnHash", txnHash,
			"receipt", receipt,
		)
		return err
	}

	p.logger.Info("preconf contract storeCommitment successful", "txnHash", txnHash)

	return nil
}
