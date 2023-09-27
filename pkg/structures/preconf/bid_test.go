package preconf

// construct test for bid
import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestBid(t *testing.T) {
	// Connect to Ganache
	key, _ := crypto.GenerateKey()

	bid, err := ConstructSignedBid(big.NewInt(10), "0xkartik", big.NewInt(2), key)
	if err != nil {
		t.Fatal(err)
	}
	address, err := bid.VerifySearcherSignature()
	t.Log(address)
	b, _ := json.Marshal(bid)
	var bid2 PreConfBid
	json.Unmarshal(b, &bid2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(hexutil.Bytes(bid2.BidHash))
	sig := hexutil.Bytes(bid2.Signature)
	t.Log(sig)

	if address.Big().Cmp(crypto.PubkeyToAddress(key.PublicKey).Big()) != 0 {
		t.Fatal("Address not same as signer")
	}
}

func TestCommitment(t *testing.T) {
	client, err := ethclient.Dial("http://54.200.76.18:8545")
	if err != nil {
		t.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	key, _ := crypto.GenerateKey()
	signer := PrivateKeySigner{key}
	bid, err := ConstructSignedBid(big.NewInt(10), "0xkadrtik", big.NewInt(2), key)
	if err != nil {
		t.Fatal(err)
	}

	b, _ := json.Marshal(bid)
	var bid2 PreConfBid
	json.Unmarshal(b, &bid2)
	commit, err := bid2.ConstructCommitment(signer)

	if err != nil {
		t.Fatal(err)
	}
	commit.VerifyBuilderSignature()
	privateKey, _ := crypto.HexToECDSA("7cea3c338ce48647725ca014a52a80b2a8eb71d184168c343150a98100439d1b")

	txn, err := commit.StoreCommitmentToDA(privateKey, "0x169c9cd14923ef3fed0e0ce98cdc71c3d6037728", client)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(txn.Hash().Hex())
}
