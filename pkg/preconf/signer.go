package preconf

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

var (
	ErrAlreadySignedBid        = errors.New("already contains hash or signature")
	ErrMissingHashSignature    = errors.New("missing hash or signature")
	ErrInvalidSignature        = errors.New("signature is not valid")
	ErrInvalidHash             = errors.New("bidhash doesn't match bid payload")
	ErrAlreadySignedCommitment = errors.New("commitment is already hashed or signed")
)

// PreConfBid represents the bid data.
// Adds blocknumber for pre-conf bid - Will need to manage
// how to reciever acts on a bid / TTL is the blocknumber
type Bid struct {
	TxnHash     string   `json:"txnHash"`
	BidAmt      *big.Int `json:"bid"`
	BlockNumber *big.Int `json:"blocknumber"`

	BidHash   []byte `json:"bidhash"` // TODO(@ckaritk): name better
	Signature []byte `json:"signature"`
}

type Commitment struct {
	Bid

	DataHash            []byte `json:"data_hash"` // TODO(@ckaritk): name better
	CommitmentSignature []byte `json:"commitment_signature"`
}

type Signer interface {
	ConstructSignedBid(string, *big.Int, *big.Int) (*Bid, error)
	ConstructCommitment(*Bid) (*Commitment, error)
	VerifyBid(*Bid) (*common.Address, error)
	VerifyCommitment(*Commitment) (*common.Address, error)
}

type privateKeySigner struct {
	privKey *ecdsa.PrivateKey
}

func NewSigner(key *ecdsa.PrivateKey) *privateKeySigner {
	return &privateKeySigner{
		privKey: key,
	}
}

func (p *privateKeySigner) ConstructSignedBid(
	txnHash string,
	bidAmt *big.Int,
	blockNumber *big.Int,
) (*Bid, error) {
	if txnHash == "" || bidAmt == nil || blockNumber == nil {
		return nil, errors.New("missing required fields")
	}

	bid := &Bid{
		BidAmt:      bidAmt,
		TxnHash:     txnHash,
		BlockNumber: blockNumber,
	}

	internalPayload := constructBidPayload(txnHash, bidAmt, blockNumber)

	bidHash, _, err := apitypes.TypedDataAndHash(internalPayload)
	if err != nil {
		return nil, err
	}

	sig, err := crypto.Sign(bidHash, p.privKey)
	if err != nil {
		return nil, err
	}

	bid.BidHash = bidHash
	bid.Signature = sig

	return bid, nil
}

func (p *privateKeySigner) ConstructCommitment(bid *Bid) (*Commitment, error) {
	_, err := p.VerifyBid(bid)
	if err != nil {
		return nil, err
	}

	commitment := &Commitment{
		Bid: *bid,
	}

	eip712Payload := constructCommitmentPayload(
		bid.TxnHash,
		bid.BidAmt,
		bid.BlockNumber,
		bid.BidHash,
		bid.Signature,
	)

	dataHash, _, err := apitypes.TypedDataAndHash(eip712Payload)
	if err != nil {
		return nil, err
	}

	sig, err := crypto.Sign(dataHash, p.privKey)
	if err != nil {
		return nil, err
	}

	commitment.DataHash = dataHash
	commitment.CommitmentSignature = sig

	return commitment, nil
}

func (p *privateKeySigner) VerifyBid(bid *Bid) (*common.Address, error) {
	if bid.BidHash == nil || bid.Signature == nil {
		return nil, ErrMissingHashSignature
	}

	return eipVerify(
		constructBidPayload(bid.TxnHash, bid.BidAmt, bid.BlockNumber),
		bid.BidHash,
		bid.Signature,
	)
}

func (p *privateKeySigner) VerifyCommitment(c *Commitment) (*common.Address, error) {
	if c.DataHash == nil || c.CommitmentSignature == nil {
		return nil, ErrMissingHashSignature
	}

	_, err := p.VerifyBid(&c.Bid)
	if err != nil {
		return nil, err
	}

	internalPayload := constructCommitmentPayload(
		c.TxnHash,
		c.BidAmt,
		c.BlockNumber,
		c.BidHash,
		c.Signature,
	)

	return eipVerify(internalPayload, c.DataHash, c.CommitmentSignature)
}

func eipVerify(
	internalPayload apitypes.TypedData,
	expectedhash []byte,
	signature []byte,
) (*common.Address, error) {
	payloadHash, _, err := apitypes.TypedDataAndHash(internalPayload)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(payloadHash, expectedhash) {
		return nil, ErrInvalidHash
	}

	pubkey, err := crypto.SigToPub(payloadHash, signature)
	if err != nil {
		return nil, err
	}

	if !crypto.VerifySignature(
		crypto.FromECDSAPub(pubkey),
		payloadHash,
		signature[:len(signature)-1],
	) {
		return nil, ErrInvalidSignature
	}

	c := crypto.PubkeyToAddress(*pubkey)

	return &c, err
}

// Constructs the EIP712 formatted bid
func constructCommitmentPayload(
	txnHash string,
	bid *big.Int,
	blockNumber *big.Int,
	bidHash []byte,
	signature []byte,
) apitypes.TypedData {
	signerData := apitypes.TypedData{
		Types: apitypes.Types{
			"PreConfCommitment": []apitypes.Type{
				{Name: "txnHash", Type: "string"},
				{Name: "bid", Type: "uint64"},
				{Name: "blockNumber", Type: "uint64"},
				{Name: "bidHash", Type: "string"},   // Hex Encoded Hash
				{Name: "signature", Type: "string"}, // Hex Encoded Signature
			},
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
			},
		},
		PrimaryType: "PreConfCommitment",
		Domain: apitypes.TypedDataDomain{
			Name:    "PreConfCommitment",
			Version: "1",
		},
		Message: apitypes.TypedDataMessage{
			"txnHash":     txnHash,
			"bid":         bid,
			"blockNumber": blockNumber,
			"bidHash":     hex.EncodeToString(bidHash),
			"signature":   hex.EncodeToString(signature),
		},
	}

	return signerData
}

// Constructs the EIP712 formatted bid
func constructBidPayload(txnHash string, bid *big.Int, blockNumber *big.Int) apitypes.TypedData {
	signerData := apitypes.TypedData{
		Types: apitypes.Types{
			"PreConfBid": []apitypes.Type{
				{Name: "txnHash", Type: "string"},
				{Name: "bid", Type: "uint64"},
				{Name: "blockNumber", Type: "uint64"},
			},
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
			},
		},
		PrimaryType: "PreConfBid",
		Domain: apitypes.TypedDataDomain{
			Name:    "PreConfBid",
			Version: "1",
		},
		Message: apitypes.TypedDataMessage{
			"txnHash":     txnHash,
			"bid":         bid,
			"blockNumber": blockNumber,
		},
	}

	return signerData
}
