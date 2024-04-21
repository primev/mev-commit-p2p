package bidderregistrycontract

import (
	"context"
	"log/slog"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	bidderregistry "github.com/primevprotocol/contracts-abi/clients/BidderRegistry"
	"github.com/primevprotocol/mev-commit/pkg/evmclient"
)

var bidderRegistryABI = func() abi.ABI {
	abi, err := abi.JSON(strings.NewReader(bidderregistry.BidderregistryMetaData.ABI))
	if err != nil {
		panic(err)
	}
	return abi
}

type Interface interface {
	// PrepayAllowanceForSpecificWindow registers a bidder with the bidder_registry contract for a specific window.
	PrepayAllowanceForSpecificWindow(ctx context.Context, amount, window *big.Int) error
	// GetAllowance returns the stake of a bidder.
	GetAllowance(ctx context.Context, address common.Address, window *big.Int) (*big.Int, error)
	// GetMinAllowance returns the minimum stake required to register as a bidder.
	GetMinAllowance(ctx context.Context) (*big.Int, error)
	// CheckBidderRegistred returns true if bidder is registered
	CheckBidderAllowance(ctx context.Context, address common.Address, window *big.Int, blocksPerWindow *big.Int) bool
	// WithdrawAllowance withdraws the stake of a bidder.
	WithdrawAllowance(ctx context.Context, window *big.Int) error
}

type bidderRegistryContract struct {
	owner                      common.Address
	bidderRegistryABI          abi.ABI
	bidderRegistryContractAddr common.Address
	client                     evmclient.Interface
	logger                     *slog.Logger
}

func New(
	owner common.Address,
	bidderRegistryContractAddr common.Address,
	client evmclient.Interface,
	logger *slog.Logger,
) Interface {
	return &bidderRegistryContract{
		owner:                      owner,
		bidderRegistryABI:          bidderRegistryABI(),
		bidderRegistryContractAddr: bidderRegistryContractAddr,
		client:                     client,
		logger:                     logger,
	}
}

func (r *bidderRegistryContract) PrepayAllowanceForSpecificWindow(ctx context.Context, amount, window *big.Int) error {
	callData, err := r.bidderRegistryABI.Pack("prepayAllowanceForSpecificWindow", window)
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
			"prepay failed for bidder registry",
			"txnHash", txnHash,
			"receipt", receipt,
		)
		return err
	}

	var bidderRegistered struct {
		Bidder        common.Address
		PrepaidAmount *big.Int
		WindowNumber  *big.Int
	}
	for _, log := range receipt.Logs {
		if len(log.Topics) > 1 {
			bidderRegistered.Bidder = common.HexToAddress(log.Topics[1].Hex())
		}

		err := r.bidderRegistryABI.UnpackIntoInterface(&bidderRegistered, "BidderRegistered", log.Data)
		if err != nil {
			r.logger.Debug("Failed to unpack event", "err", err)
			continue
		}
		r.logger.Info("bidder registered", "address", bidderRegistered.Bidder, "prepaidAmount", bidderRegistered.PrepaidAmount.String(), "windowNumber", bidderRegistered.WindowNumber.Int64())
	}

	r.logger.Info("prepay successful for bidder registry", "txnHash", txnHash, "bidder", bidderRegistered.Bidder)

	return nil
}

func (r *bidderRegistryContract) GetAllowance(
	ctx context.Context,
	address common.Address,
	window *big.Int,
) (*big.Int, error) {
	callData, err := r.bidderRegistryABI.Pack("getAllowance", address, window)
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

	results, err := r.bidderRegistryABI.Unpack("getAllowance", result)
	if err != nil {
		r.logger.Error("error unpacking result", "error", err)
		return nil, err
	}

	return abi.ConvertType(results[0], new(big.Int)).(*big.Int), nil
}

func (r *bidderRegistryContract) GetMinAllowance(ctx context.Context) (*big.Int, error) {
	callData, err := r.bidderRegistryABI.Pack("minAllowance")
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

	results, err := r.bidderRegistryABI.Unpack("minAllowance", result)
	if err != nil {
		r.logger.Error("error unpacking result", "error", err)
		return nil, err
	}

	return abi.ConvertType(results[0], new(big.Int)).(*big.Int), nil
}

func (r *bidderRegistryContract) WithdrawAllowance(ctx context.Context, window *big.Int) error {
	callData, err := r.bidderRegistryABI.Pack("withdrawBidderAmountFromWindow", r.owner, window)
	if err != nil {
		r.logger.Error("error packing call data", "error", err)
		return err
	}

	txnHash, err := r.client.Send(ctx, &evmclient.TxRequest{
		To:       &r.bidderRegistryContractAddr,
		CallData: callData,
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
			"withdraw failed for bidder registry",
			"txnHash", txnHash,
			"receipt", receipt,
		)
		return err
	}

	var bidderWithdrawn struct {
		Bidder common.Address
		Amount *big.Int
		Window *big.Int
	}

	for _, log := range receipt.Logs {
		if len(log.Topics) > 1 {
			bidderWithdrawn.Bidder = common.HexToAddress(log.Topics[1].Hex())
		}

		err := r.bidderRegistryABI.UnpackIntoInterface(&bidderWithdrawn, "BidderWithdrawn", log.Data)
		if err != nil {
			r.logger.Debug("Failed to unpack event", "err", err)
			continue
		}
		r.logger.Info("bidder withdrawn", "address", bidderWithdrawn.Bidder, "withdrawn", bidderWithdrawn.Amount.Uint64(), "windowNumber", bidderWithdrawn.Window.Int64())
	}

	r.logger.Info("withdraw successful for bidder registry", "txnHash", txnHash, "bidder", bidderWithdrawn.Bidder)

	return nil
}

func (r *bidderRegistryContract) CheckBidderAllowance(
	ctx context.Context,
	address common.Address,
	window *big.Int,
	blocksPerWindow *big.Int,
) bool {
	minStake, err := r.GetMinAllowance(ctx)
	if err != nil {
		r.logger.Error("error getting min stake", "error", err)
		return false
	}

	stake, err := r.GetAllowance(ctx, address, window)
	if err != nil {
		r.logger.Error("error getting stake", "error", err)
		return false
	}
	r.logger.Info("checking bidder allowance",
		"stake", stake.Uint64(),
		"blocksPerWindow", blocksPerWindow.Uint64(),
		"minStake", minStake.Uint64(),
		"window", window.Uint64(),
		"address", address.Hex(),
	)
	return (stake.Div(stake, blocksPerWindow)).Cmp(minStake) >= 0
}
