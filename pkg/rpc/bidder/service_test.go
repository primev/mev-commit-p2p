package bidderapi_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/bufbuild/protovalidate-go"
	"github.com/ethereum/go-ethereum/common"
	bidderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/bidderapi/v1"
	bidderapi "github.com/primevprotocol/mev-commit/pkg/rpc/bidder"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
	"github.com/primevprotocol/mev-commit/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type bid struct {
	txHex    string
	amount   *big.Int
	blockNum *big.Int
}

type testSender struct {
	bids         []bid
	noOfPreconfs int
}

func (s *testSender) SendBid(
	ctx context.Context,
	txHex string,
	amount *big.Int,
	blockNum *big.Int,
) (chan *preconfsigner.PreConfirmation, error) {
	s.bids = append(s.bids, bid{
		txHex:    txHex,
		amount:   amount,
		blockNum: blockNum,
	})

	preconfs := make(chan *preconfsigner.PreConfirmation, s.noOfPreconfs)
	for i := 0; i < s.noOfPreconfs; i++ {
		preconfs <- &preconfsigner.PreConfirmation{
			Bid: preconfsigner.Bid{
				TxHash:      txHex,
				BidAmt:      amount,
				BlockNumber: blockNum,
				Digest:      []byte("digest"),
				Signature:   []byte("signature"),
			},
			Digest:          []byte("digest"),
			Signature:       []byte("signature"),
			ProviderAddress: common.HexToAddress(fmt.Sprintf("0x0000%d", i)),
		}
	}

	close(preconfs)

	return preconfs, nil
}

type testRegistryContract struct {
	allowance    *big.Int
	minAllowance *big.Int
}

func (t *testRegistryContract) PrepayAllowance(ctx context.Context, amount *big.Int) error {
	t.allowance = amount
	return nil
}

func (t *testRegistryContract) GetAllowance(ctx context.Context, address common.Address) (*big.Int, error) {
	return t.allowance, nil
}

func (t *testRegistryContract) GetMinAllowance(ctx context.Context) (*big.Int, error) {
	return t.minAllowance, nil
}

func (t *testRegistryContract) CheckBidderAllowance(ctx context.Context, address common.Address) bool {
	return t.allowance.Cmp(t.minAllowance) > 0
}

func startServer(t *testing.T) bidderapiv1.BidderClient {
	buffer := 101024 * 1024
	lis := bufconn.Listen(buffer)

	logger := util.NewTestLogger(os.Stdout)
	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("error creating validator: %v", err)
	}

	owner := common.HexToAddress("0x00001")
	registryContract := &testRegistryContract{minAllowance: big.NewInt(100000000000000000)}
	sender := &testSender{noOfPreconfs: 2}

	srvImpl := bidderapi.NewService(
		sender,
		owner,
		registryContract,
		validator,
		logger,
	)

	baseServer := grpc.NewServer()
	bidderapiv1.RegisterBidderServer(baseServer, srvImpl)
	go func() {
		if err := baseServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()

	conn, err := grpc.DialContext(context.TODO(), "",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("error connecting to server: %v", err)
	}

	t.Cleanup(func() {
		err := lis.Close()
		if err != nil {
			log.Printf("error closing listener: %v", err)
		}
		baseServer.Stop()
	})

	client := bidderapiv1.NewBidderClient(conn)

	return client
}

func TestAllowanceHandling(t *testing.T) {
	t.Parallel()

	client := startServer(t)

	t.Run("prepay", func(t *testing.T) {
		type testCase struct {
			amount string
			err    string
		}

		for _, tc := range []testCase{
			{
				amount: "",
				err:    "amount must be a valid integer",
			},
			{
				amount: "0000000000000000000",
				err:    "amount must be a valid integer",
			},
			{
				amount: "asdf",
				err:    "amount must be a valid integer",
			},
			{
				amount: "1000000000000000000",
				err:    "",
			},
		} {
			allowance, err := client.PrepayAllowance(context.Background(), &bidderapiv1.PrepayRequest{Amount: tc.amount})
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("expected error prepaying allowance")
				}
			} else {
				if err != nil {
					t.Fatalf("error prepaying allowance: %v", err)
				}
				if allowance.Amount != tc.amount {
					t.Fatalf("expected amount to be %v, got %v", tc.amount, allowance.Amount)
				}
			}
		}
	})

	t.Run("get allowance", func(t *testing.T) {
		allowance, err := client.GetAllowance(context.Background(), &bidderapiv1.EmptyMessage{})
		if err != nil {
			t.Fatalf("error getting allowance: %v", err)
		}
		if allowance.Amount != "1000000000000000000" {
			t.Fatalf("expected amount to be 1000000000000000000, got %v", allowance.Amount)
		}
	})

	t.Run("get min allowance", func(t *testing.T) {
		allowance, err := client.GetMinAllowance(context.Background(), &bidderapiv1.EmptyMessage{})
		if err != nil {
			t.Fatalf("error getting min allowance: %v", err)
		}
		if allowance.Amount != "100000000000000000" {
			t.Fatalf("expected amount to be 100000000000000000, got %v", allowance.Amount)
		}
	})
}

func TestSendBid(t *testing.T) {
	t.Parallel()

	client := startServer(t)

	type testCase struct {
		name     string
		txHexs   []string
		amount   string
		blockNum int64
		err      string
	}

	fmt.Println(common.HexToHash("0x0000ab").Hex()[2:])

	for _, tc := range []testCase{
		{
			name:     "invalid tx hex",
			txHexs:   []string{"asdf"},
			amount:   "1000000000000000000",
			blockNum: 1,
			err:      "tx_hashes must be a valid array of transaction hashes",
		},
		{
			name:     "no txns",
			txHexs:   nil,
			amount:   "1000000000000000000",
			blockNum: 1,
			err:      "tx_hashes must be a valid array of transaction hashes",
		},
		{
			name:     "invalid amount",
			txHexs:   []string{common.HexToHash("0x0000ab").Hex()[2:]},
			amount:   "000000000000000000",
			blockNum: 1,
			err:      "amount must be a valid integer",
		},
		{
			name:     "invalid block number",
			txHexs:   []string{common.HexToHash("0x0000ab").Hex()[2:]},
			amount:   "1000000000000000000",
			blockNum: 0,
			err:      "block_number must be a valid integer",
		},
		{
			name:     "success",
			txHexs:   []string{common.HexToHash("0x0000ab").Hex()[2:]},
			amount:   "1000000000000000000",
			blockNum: 1,
			err:      "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rcv, err := client.SendBid(context.Background(), &bidderapiv1.Bid{
				TxHashes:    tc.txHexs,
				Amount:      tc.amount,
				BlockNumber: tc.blockNum,
			})
			if err != nil {
				t.Fatalf("error sending bid: %v", err)
			}

			if tc.err != "" {
				_, err := rcv.Recv()
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("expected error sending bid %s, got %v", tc.err, err)
				}
			} else {
				count := 0
				for {
					_, err := rcv.Recv()
					if err != nil {
						if errors.Is(err, io.EOF) {
							break
						}
						t.Fatalf("error receiving preconfs: %v", err)
					}
					count++
				}
				if count != 2 {
					t.Fatalf("expected 2 preconfs, got %v", count)
				}
			}
		})
	}
}
