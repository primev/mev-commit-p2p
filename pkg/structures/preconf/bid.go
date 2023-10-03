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
type PreConfBid struct { // Adds blocknumber for pre-conf bid - Will need to manage how to reciever acts on a bid / TTL is the blocknumber
	TxnHash     string   `json:"txnHash"`
	Bid         *big.Int `json:"bid"`
	Blocknumber *big.Int `json:"blocknumber"`

	BidHash   []byte `json:"bidhash"` // TODO(@ckaritk): name better
	Signature []byte `json:"signature"`
}

type PreconfCommitment struct {
	PreConfBid

	DataHash            []byte `json:"data_hash"` // TODO(@ckaritk): name better
	CommitmentSignature []byte `json:"commitment_signature"`
}

type Signer interface {
	Sign([]byte) ([]byte, error)
}

type PrivateKeySigner struct {
	PrivKey *ecdsa.PrivateKey
}

func (p PrivateKeySigner) Sign(digest []byte) ([]byte, error) {
	return crypto.Sign(digest, p.PrivKey)
}

func (p *PreconfCommitment) VerifyBuilderSignature() (*common.Address, error) {
	if p.DataHash == nil || p.CommitmentSignature == nil {
		return nil, ErrMissingHashSignature
	}

	internalPayload := constructCommitmentPayload(p.TxnHash, p.Bid, p.Blocknumber, p.BidHash, p.Signature)

	return eipVerify(internalPayload, p.DataHash, p.CommitmentSignature)
}

func eipVerify(internalPayload apitypes.TypedData, expectedhash []byte, signature []byte) (*common.Address, error) {
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

	if !crypto.VerifySignature(crypto.FromECDSAPub(pubkey), payloadHash, signature[:len(signature)-1]) {
		return nil, ErrInvalidSignature
	}

	c := crypto.PubkeyToAddress(*pubkey)

	return &c, err
}

func (p *PreConfBid) BidOriginator() (*common.Address, *ecdsa.PublicKey, error) {
	_, err := p.VerifySearcherSignature()
	if err != nil {
		return nil, nil, err
	}

	pubkey, err := crypto.SigToPub(p.BidHash, p.Signature)
	if err != nil {
		return nil, nil, err
	}

	address := crypto.PubkeyToAddress(*pubkey)

	return &address, pubkey, nil
}

func (p *PreconfCommitment) CommitmentOriginator() (*common.Address, *ecdsa.PublicKey, error) {
	_, err := p.VerifyBuilderSignature()
	if err != nil {
		return nil, nil, err
	}

	pubkey, err := crypto.SigToPub(p.DataHash, p.CommitmentSignature)
	if err != nil {
		return nil, nil, err
	}

	address := crypto.PubkeyToAddress(*pubkey)

	return &address, pubkey, nil
}

func ConstructCommitment(p PreConfBid, signer Signer) (*PreconfCommitment, error) {
	_, err := p.VerifySearcherSignature()
	if err != nil {
		return nil, err
	}
	commitment := PreconfCommitment{
		PreConfBid: p,
	}

	eip712Payload := constructCommitmentPayload(p.TxnHash, p.Bid, p.Blocknumber, p.BidHash, p.Signature)

	dataHash, _, err := apitypes.TypedDataAndHash(eip712Payload)
	if err != nil {
		return nil, err
	}

	sig, err := signer.Sign(dataHash)
	if err != nil {
		return nil, err
	}

	commitment.DataHash = dataHash
	commitment.CommitmentSignature = sig

	return &commitment, nil
}

// Returns a PreConfBid Object with an EIP712 signature of the payload
func ConstructSignedBid(bidamt *big.Int, txnhash string, blocknumber *big.Int, signer Signer) (*PreConfBid, error) {
	bid := PreConfBid{
		Bid:         bidamt,
		TxnHash:     txnhash,
		Blocknumber: blocknumber,
	}

	internalPayload := constructBidPayload(txnhash, bidamt, blocknumber)

	bidHash, _, err := apitypes.TypedDataAndHash(internalPayload)
	if err != nil {
		return nil, err
	}

	sig, err := signer.Sign(bidHash)
	if err != nil {
		return nil, err
	}

	bid.BidHash = bidHash
	bid.Signature = sig

	return &bid, nil
}

// Verifies the bid
func (p *PreConfBid) VerifySearcherSignature() (*common.Address, error) {
	if p.BidHash == nil || p.Signature == nil {
		return nil, ErrMissingHashSignature
	}

	return eipVerify(constructBidPayload(p.TxnHash, p.Bid, p.Blocknumber), p.BidHash, p.Signature)
}

// Constructs the EIP712 formatted bid
func constructCommitmentPayload(txnHash string, bid *big.Int, blockNumber *big.Int, bidHash []byte, signature []byte) apitypes.TypedData {
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
