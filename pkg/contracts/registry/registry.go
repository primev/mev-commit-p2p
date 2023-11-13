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
		To:        &r.registryContractAddr,
		Nonce:     nonce,
		Data:      callData,
		Value:     amount,
		Gas:       uint64(gasPrice.Int64()),
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

	err = r.client.SendTransaction(ctx, signedTxn)
	if err != nil {
		return err
	}

	r.logger.Info("sent txn", "txnData", txnData)

	receipt, err := bind.WaitMined(ctx, r.client, txnData)
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
	var address [32]byte
	copy(address[:], r.owner.Bytes())

	callData, err := r.registryABI.Pack("providerStakes", address)
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

	var stake *big.Int
	err = r.registryABI.UnpackIntoInterface(&stake, "providerStakes", result)
	if err != nil {
		r.logger.Error("error unpacking result", "error", err)
		return nil, err
	}

	r.logger.Info("got stake", "address", address, "stake", stake)

	return stake, nil
}
