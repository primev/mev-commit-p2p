package mockevmclient

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/primevprotocol/mev-commit/pkg/evmclient"
)

type Option func(*mockEvmClient)

func New(opts ...Option) *mockEvmClient {
	m := &mockEvmClient{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func WithSendFunc(
	f func(ctx context.Context, req *evmclient.TxRequest) (common.Hash, error),
) Option {
	return func(m *mockEvmClient) {
		m.SendFunc = f
	}
}

func WithWaitForReceiptFunc(
	f func(ctx context.Context, txnHash common.Hash) (*types.Receipt, error),
) Option {
	return func(m *mockEvmClient) {
		m.WaitForReceiptFunc = f
	}
}

func WithCallFunc(
	f func(ctx context.Context, req *evmclient.TxRequest) ([]byte, error),
) Option {
	return func(m *mockEvmClient) {
		m.CallFunc = f
	}
}

type mockEvmClient struct {
	SendFunc           func(ctx context.Context, req *evmclient.TxRequest) (common.Hash, error)
	WaitForReceiptFunc func(ctx context.Context, txnHash common.Hash) (*types.Receipt, error)
	CallFunc           func(ctx context.Context, req *evmclient.TxRequest) ([]byte, error)
}

func (m *mockEvmClient) Send(
	ctx context.Context,
	req *evmclient.TxRequest,
) (common.Hash, error) {
	return m.SendFunc(ctx, req)
}

func (m *mockEvmClient) WaitForReceipt(
	ctx context.Context,
	txnHash common.Hash,
) (*types.Receipt, error) {
	return m.WaitForReceiptFunc(ctx, txnHash)
}

func (m *mockEvmClient) Call(ctx context.Context, req *evmclient.TxRequest) ([]byte, error) {
	return m.CallFunc(ctx, req)
}
