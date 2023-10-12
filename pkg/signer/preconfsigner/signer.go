package preconfsigner

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
	ErrAlreadySignedBid             = errors.New("already contains hash or signature")
	ErrMissingHashSignature         = errors.New("missing hash or signature")
	ErrInvalidSignature             = errors.New("signature is not valid")
	ErrInvalidHash                  = errors.New("bidhash doesn't match bid payload")
	ErrAlreadySignedPreConfirmation = errors.New("preConfirmation is already hashed or signed")
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

type PreConfirmation struct {
	Bid

	PreconfirmationDigest    []byte `json:"preconfirmation_digest"` // TODO(@ckaritk): name better
	PreConfirmationSignature []byte `json:"preConfirmation_signature"`
}

type Signer interface {
	ConstructSignedBid(string, *big.Int, *big.Int) (*Bid, error)
	ConstructPreConfirmation(*Bid) (*PreConfirmation, error)
	VerifyBid(*Bid) (*common.Address, error)
	VerifyPreConfirmation(*PreConfirmation) (*common.Address, error)
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

func (p *privateKeySigner) ConstructPreConfirmation(bid *Bid) (*PreConfirmation, error) {
	_, err := p.VerifyBid(bid)
	if err != nil {
		return nil, err
	}

	preConfirmation := &PreConfirmation{
		Bid: *bid,
	}

	eip712Payload := constructPreConfirmationPayload(
		bid.TxnHash,
		bid.BidAmt,
		bid.BlockNumber,
		bid.BidHash,
		bid.Signature,
	)

	preconfirmationDigest, _, err := apitypes.TypedDataAndHash(eip712Payload)
	if err != nil {
		return nil, err
	}

	sig, err := crypto.Sign(preconfirmationDigest, p.privKey)
	if err != nil {
		return nil, err
	}

	preConfirmation.PreconfirmationDigest = preconfirmationDigest
	preConfirmation.PreConfirmationSignature = sig

	return preConfirmation, nil
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

func (p *privateKeySigner) VerifyPreConfirmation(c *PreConfirmation) (*common.Address, error) {
	if c.PreconfirmationDigest == nil || c.PreConfirmationSignature == nil {
		return nil, ErrMissingHashSignature
	}

	_, err := p.VerifyBid(&c.Bid)
	if err != nil {
		return nil, err
	}

	internalPayload := constructPreConfirmationPayload(
		c.TxnHash,
		c.BidAmt,
		c.BlockNumber,
		c.BidHash,
		c.Signature,
	)

	return eipVerify(internalPayload, c.PreconfirmationDigest, c.PreConfirmationSignature)
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
func constructPreConfirmationPayload(
	txnHash string,
	bid *big.Int,
	blockNumber *big.Int,
	bidHash []byte,
	signature []byte,
) apitypes.TypedData {
	signerData := apitypes.TypedData{
		Types: apitypes.Types{
			"PreConfPreConfirmation": []apitypes.Type{
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
		PrimaryType: "PreConfPreConfirmation",
		Domain: apitypes.TypedDataDomain{
			Name:    "PreConfPreConfirmation",
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
