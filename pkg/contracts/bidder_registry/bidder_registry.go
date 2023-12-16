package bidderregistrycontract

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

var bidderRegistryABI = func() abi.ABI {
	abi, err := abi.JSON(strings.NewReader(contractsabi.UserRegistryJson))
	if err != nil {
		panic(err)
	}
	return abi
}

type Interface interface {
	// RegisterBidder registers a provider with the provider_registry contract.
	RegisterBidder(ctx context.Context, amount *big.Int) error
	// GetStake returns the stake of a provider.
	GetStake(ctx context.Context, address common.Address) (*big.Int, error)
	// GetMinStake returns the minimum stake required to register as a provider.
	GetMinStake(ctx context.Context) (*big.Int, error)
	// CheckBidderRegistred returns true if bidder is registered
	CheckBidderRegistered(ctx context.Context, address common.Address) bool
}

type bidderRegistryContract struct {
	bidderRegistryABI          abi.ABI
	bidderRegistryContractAddr common.Address
	client                     evmclient.Interface
	logger                     *slog.Logger
}

func New(
	bidderRegistryContractAddr common.Address,
	client evmclient.Interface,
	logger *slog.Logger,
) Interface {
	return &bidderRegistryContract{
		bidderRegistryABI:          bidderRegistryABI(),
		bidderRegistryContractAddr: bidderRegistryContractAddr,
		client:                     client,
		logger:                     logger,
	}
}

func (r *bidderRegistryContract) RegisterBidder(ctx context.Context, amount *big.Int) error {
	callData, err := r.bidderRegistryABI.Pack("registerAndStake")
	if err != nil {
		r.logger.Error("error packing call data", "error", err)
		return err
	}

	txnHash, err := r.client.Send(ctx, &evmclient.TxRequest{
		To:       &r.bidderRegistryContractAddr,
		CallData: callData,
		Value:    amount,
	})
	if err != nil {
		return err
	}

	receipt, err := r.client.WaitForReceipt(ctx, txnHash)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		r.logger.Error(
			"provider_registry contract registerAndStake failed",
			"txnHash", txnHash,
			"receipt", receipt,
		)
		return err
	}

	r.logger.Info("provider_registry contract registerAndStake successful", "txnHash", txnHash)

	return nil
}

func (r *bidderRegistryContract) GetStake(
	ctx context.Context,
	address common.Address,
) (*big.Int, error) {
	callData, err := r.bidderRegistryABI.Pack("checkStake", address)
	if err != nil {
		r.logger.Error("error packing call data", "error", err)
		return nil, err
	}

	result, err := r.client.Call(ctx, &evmclient.TxRequest{
		To:       &r.bidderRegistryContractAddr,
		CallData: callData,
	})
	if err != nil {
		return nil, err
	}

	results, err := r.bidderRegistryABI.Unpack("checkStake", result)
	if err != nil {
		r.logger.Error("error unpacking result", "error", err)
		return nil, err
	}

	return abi.ConvertType(results[0], new(big.Int)).(*big.Int), nil
}

func (r *bidderRegistryContract) GetMinStake(ctx context.Context) (*big.Int, error) {
	callData, err := r.bidderRegistryABI.Pack("minStake")
	if err != nil {
		r.logger.Error("error packing call data", "error", err)
		return nil, err
	}

	result, err := r.client.Call(ctx, &evmclient.TxRequest{
		To:       &r.bidderRegistryContractAddr,
		CallData: callData,
	})
	if err != nil {
		return nil, err
	}

	results, err := r.bidderRegistryABI.Unpack("minStake", result)
	if err != nil {
		r.logger.Error("error unpacking result", "error", err)
		return nil, err
	}

	return abi.ConvertType(results[0], new(big.Int)).(*big.Int), nil
}

func (r *bidderRegistryContract) CheckBidderRegistered(
	ctx context.Context,
	address common.Address,
) bool {

	minStake, err := r.GetMinStake(ctx)
	if err != nil {
		r.logger.Error("error getting min stake", "error", err)
		return false
	}

	stake, err := r.GetStake(ctx, address)
	if err != nil {
		r.logger.Error("error getting stake", "error", err)
		return false
	}

	return stake.Cmp(minStake) >= 0
}
