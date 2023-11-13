package registrycontract

import (
	"context"
	"crypto/ecdsa"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Interface interface {
	// RegisterProvider registers a provider with the registry contract.
	RegisterProvider(ctx context.Context, amount *big.Int) error
	// GetStake returns the stake of a provider.
	GetStake(ctx context.Context) (*big.Int, error)
}

type registryContract struct {
	registryABI          abi.ABI
	registryContractAddr common.Address
	client               *ethclient.Client
	owner                common.Address
	signer               *ecdsa.PrivateKey
	logger               *slog.Logger
}

func New(
	owner common.Address,
	registryContractAddr common.Address,
	registryABI abi.ABI,
	client *ethclient.Client,
	signer *ecdsa.PrivateKey,
	logger *slog.Logger,
) Interface {
	return &registryContract{
		registryABI:          registryABI,
		registryContractAddr: registryContractAddr,
		client:               client,
		owner:                owner,
		signer:               signer,
		logger:               logger,
	}
}

func (r *registryContract) RegisterProvider(ctx context.Context, amount *big.Int) error {
	callData, err := r.registryABI.Pack("registerAndStake")
	if err != nil {
		return err
	}

	nonce, err := r.client.PendingNonceAt(ctx, r.owner)
	if err != nil {
		return err
	}

	chainID, err := r.client.NetworkID(ctx)
	if err != nil {
		return err
	}

	r.logger.Info("chainID", "chainID", chainID, "nonce", nonce)

	gasPrice, err := r.client.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}

	gasTipCap, err := r.client.SuggestGasTipCap(ctx)
	if err != nil {
		return err
	}

	gas := big.NewInt(10000000)
	// gasTipCap := big.NewInt(0)

	gasFeeCap := new(big.Int).Add(gasPrice, gasTipCap)

	txnData := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		To:        &r.registryContractAddr,
		Nonce:     nonce,
		Data:      callData,
		Value:     amount,
		Gas:       uint64(gas.Int64()),
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
	})

	signedTxn, err := types.SignTx(
		txnData,
		types.LatestSignerForChainID(chainID),
		r.signer,
	)
	if err != nil {
		return err
	}

	r.logger.Info("signed txn", "txnData", signedTxn)

	err = r.client.SendTransaction(ctx, signedTxn)
	if err != nil {
		return err
	}

	// data, err := signedTxn.MarshalBinary()
	// if err != nil {
	// 	return err
	// }
	// rawTxHex := hexutil.Encode(data)

	// r.logger.Info("sent txn", "txnData", txnData, "txHash", signedTxn.Hash(), "rawTxHex", rawTxHex)

	receipt, err := bind.WaitMined(ctx, r.client, signedTxn)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return err
	}

	r.logger.Info("registry contract registerAndStake successful", "txnData", txnData)

	return nil
}

func (r *registryContract) GetStake(ctx context.Context) (*big.Int, error) {
	callData, err := r.registryABI.Pack("checkStake", r.owner)
	if err != nil {
		r.logger.Error("error packing call data", "error", err)
		return nil, err
	}

	msg := ethereum.CallMsg{
		From: r.owner,
		To:   &r.registryContractAddr,
		Data: callData,
	}

	result, err := r.client.CallContract(ctx, msg, nil)
	if err != nil {
		r.logger.Error("error calling contract", "error", err)
		return nil, err
	}

	results, err := r.registryABI.Unpack("checkStake", result)
	if err != nil {
		r.logger.Error("error unpacking result", "error", err)
		return nil, err
	}

	return abi.ConvertType(results[0], new(big.Int)).(*big.Int), nil
}
