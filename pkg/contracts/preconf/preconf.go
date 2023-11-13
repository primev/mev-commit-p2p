package preconfcontract

import (
	"context"
	"crypto/ecdsa"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Interface interface {
	StoreCommitment(
		ctx context.Context,
		bid *big.Int,
		blockNumber uint64,
		txHash string,
		commitmentHash string,
		bidSignature []byte,
		commitmentSignature []byte,
	) error
}

type preconfContract struct {
	preconfABI          abi.ABI
	preconfContractAddr common.Address
	client              *ethclient.Client
	owner               common.Address
	signer              *ecdsa.PrivateKey
	logger              *slog.Logger
}

func New(
	owner common.Address,
	preconfContractAddr common.Address,
	preconfABI abi.ABI,
	client *ethclient.Client,
	signer *ecdsa.PrivateKey,
	logger *slog.Logger,
) Interface {
	return &preconfContract{
		preconfABI:          preconfABI,
		preconfContractAddr: preconfContractAddr,
		client:              client,
		owner:               owner,
		signer:              signer,
		logger:              logger,
	}
}

func (p *preconfContract) StoreCommitment(
	ctx context.Context,
	bid *big.Int,
	blockNumber uint64,
	txHash string,
	commitmentHash string,
	bidSignature []byte,
	commitmentSignature []byte,
) error {

	callData, err := p.preconfABI.Pack(
		"storeCommitment",
		uint64(bid.Int64()),
		blockNumber,
		txHash,
		commitmentHash,
		bidSignature,
		commitmentSignature,
	)
	if err != nil {
		p.logger.Error("preconf contract storeCommitment pack error", "err", err)
		return err
	}

	nonce, err := p.client.PendingNonceAt(ctx, p.owner)
	if err != nil {
		return err
	}

	chainID, err := p.client.NetworkID(ctx)
	if err != nil {
		return err
	}

	// gasPrice, err := p.client.SuggestGasPrice(ctx)
	// if err != nil {
	// 	return err
	// }

	// gasTipCap, err := p.client.SuggestGasTipCap(ctx)
	// if err != nil {
	// 	return err
	// }

	gasPrice := big.NewInt(50000)
	gasTipCap := big.NewInt(50000)

	gasFeeCap := new(big.Int).Add(gasPrice, gasTipCap)

	txnData := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		To:        &p.preconfContractAddr,
		Nonce:     nonce,
		Data:      callData,
		Gas:       uint64(gasPrice.Int64()),
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
	})

	signedTxn, err := types.SignTx(
		txnData,
		types.LatestSignerForChainID(chainID),
		p.signer,
	)
	if err != nil {
		return err
	}

	err = p.client.SendTransaction(ctx, signedTxn)
	if err != nil {
		return err
	}

	p.logger.Info("sent txn", "txnData", txnData)

	receipt, err := bind.WaitMined(ctx, p.client, txnData)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return err
	}

	p.logger.Info("preconf contract storeCommitment successful", "txnData", txnData)

	return nil
}
