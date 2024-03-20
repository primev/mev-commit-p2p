package preconfcontract

import (
	"context"
	"log/slog"
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

func (p *preconfContract) StoreEncryptedCommitment(
	ctx context.Context,
	commitmentDigest []byte,
	commitmentSignature []byte,
) error {

	callData, err := p.preconfABI.Pack(
		"storeEncryptedCommitment",
		[32]byte(commitmentDigest),
		commitmentSignature,
	)
	if err != nil {
		p.logger.Error("preconf contract storeEncryptedCommitment pack error", "err", err)
		return err
	}

	txnHash, err := p.client.Send(ctx, &evmclient.TxRequest{
		To:       &p.preconfContractAddr,
		CallData: callData,
	})
	if err != nil {
		return err
	}

	p.logger.Info("preconf contract storeEncryptedCommitment successful", "txnHash", txnHash)

	return nil
}
