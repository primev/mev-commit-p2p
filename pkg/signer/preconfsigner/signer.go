package preconfsigner

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
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
	TxHash      string   `json:"txn_hash"`
	BidAmt      *big.Int `json:"bid_amt"`
	BlockNumber *big.Int `json:"block_number"`

	Digest    []byte `json:"bid_digest"` // TODO(@ckaritk): name better
	Signature []byte `json:"bid_signature"`
}

type PreConfirmation struct {
	Bid Bid `json:"bid"`

	Digest    []byte `json:"digest"` // TODO(@ckaritk): name better
	Signature []byte `json:"signature"`
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
	txHash string,
	bidAmt *big.Int,
	blockNumber *big.Int,
) (*Bid, error) {
	if txHash == "" || bidAmt == nil || blockNumber == nil {
		return nil, errors.New("missing required fields")
	}

	bid := &Bid{
		BidAmt:      bidAmt,
		TxHash:      txHash,
		BlockNumber: blockNumber,
	}

	bidHash, err := GetBidHash(bid)
	if err != nil {
		return nil, err
	}

	sig, err := crypto.Sign(bidHash, p.privKey)
	if err != nil {
		return nil, err
	}

	bid.Digest = bidHash
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

	preConfirmationHash, err := GetPreConfirmationHash(preConfirmation)
	if err != nil {
		return nil, err
	}

	sig, err := crypto.Sign(preConfirmationHash, p.privKey)
	if err != nil {
		return nil, err
	}

	preConfirmation.Digest = preConfirmationHash
	preConfirmation.Signature = sig

	return preConfirmation, nil
}

func (p *privateKeySigner) VerifyBid(bid *Bid) (*common.Address, error) {
	if bid.Digest == nil || bid.Signature == nil {
		return nil, ErrMissingHashSignature
	}

	return eipVerify(
		constructBidPayload(bid.TxHash, bid.BidAmt, bid.BlockNumber),
		bid.Digest,
		bid.Signature,
	)
}

func (p *privateKeySigner) VerifyPreConfirmation(c *PreConfirmation) (*common.Address, error) {
	if c.Digest == nil || c.Signature == nil {
		return nil, ErrMissingHashSignature
	}

	_, err := p.VerifyBid(&c.Bid)
	if err != nil {
		return nil, err
	}

	internalPayload := constructPreConfirmationPayload(
		c.Bid.TxHash,
		c.Bid.BidAmt,
		c.Bid.BlockNumber,
		c.Bid.Digest,
		c.Bid.Signature,
	)

	return eipVerify(internalPayload, c.Digest, c.Signature)
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

// GetBidHash returns the hash of the bid message. This is done manually to match the
// Solidity implementation. If the types change, this will need to be updated.
func GetBidHash(bid *Bid) ([]byte, error) {
	// DOMAIN_SEPARATOR_BID
	var (
		domainTypeHash = crypto.Keccak256Hash(
			[]byte("EIP712Domain(string name,string version)"),
		)
		nameHash           = crypto.Keccak256Hash([]byte("PreConfBid"))
		versionHash        = crypto.Keccak256Hash([]byte("1"))
		domainSeparatorBid = crypto.Keccak256Hash(
			append(append(domainTypeHash.Bytes(), nameHash.Bytes()...), versionHash.Bytes()...),
		)
	)

	// EIP712_MESSAGE_TYPEHASH
	eip712MessageTypeHash := crypto.Keccak256Hash(
		[]byte("PreConfBid(string txnHash,uint64 bid,uint64 blockNumber)"),
	)

	// Convert the txnHash to a byte array and hash it
	txnHashHash := crypto.Keccak256Hash([]byte(bid.TxHash))

	// Encode values similar to Solidity's abi.encode
	data := append(eip712MessageTypeHash.Bytes(), txnHashHash.Bytes()...)
	data = append(data, math.U256Bytes(bid.BidAmt)...)
	data = append(data, math.U256Bytes(bid.BlockNumber)...)
	dataHash := crypto.Keccak256Hash(data)

	rawData := append([]byte("\x19\x01"), append(domainSeparatorBid.Bytes(), dataHash.Bytes()...)...)
	// Create the final hash
	return crypto.Keccak256Hash(rawData).Bytes(), nil
}

// GetPreConfirmationHash returns the hash of the preconfirmation message. This is done manually to match the
// Solidity implementation. If the types change, this will need to be updated.
func GetPreConfirmationHash(c *PreConfirmation) ([]byte, error) {
	// DOMAIN_SEPARATOR_BID
	var (
		domainTypeHash = crypto.Keccak256Hash(
			[]byte("EIP712Domain(string name,string version)"),
		)
		nameHash           = crypto.Keccak256Hash([]byte("PreConfCommitment"))
		versionHash        = crypto.Keccak256Hash([]byte("1"))
		domainSeparatorBid = crypto.Keccak256Hash(
			append(append(domainTypeHash.Bytes(), nameHash.Bytes()...), versionHash.Bytes()...),
		)
	)

	// EIP712_MESSAGE_TYPEHASH
	eip712MessageTypeHash := crypto.Keccak256Hash(
		[]byte("PreConfCommitment(string txnHash,uint64 bid,uint64 blockNumber,string bidHash,string signature)"),
	)

	// Convert the txnHash to a byte array and hash it
	txnHashHash := crypto.Keccak256Hash([]byte(c.Bid.TxHash))
	bidDigestHash := crypto.Keccak256Hash([]byte(hex.EncodeToString(c.Bid.Digest)))
	bidSigHash := crypto.Keccak256Hash([]byte(hex.EncodeToString(c.Bid.Signature)))

	// Encode values similar to Solidity's abi.encode
	data := append(eip712MessageTypeHash.Bytes(), txnHashHash.Bytes()...)
	data = append(data, math.U256Bytes(c.Bid.BidAmt)...)
	data = append(data, math.U256Bytes(c.Bid.BlockNumber)...)
	data = append(data, bidDigestHash.Bytes()...)
	data = append(data, bidSigHash.Bytes()...)
	dataHash := crypto.Keccak256Hash(data)

	rawData := append([]byte("\x19\x01"), append(domainSeparatorBid.Bytes(), dataHash.Bytes()...)...)
	// Create the final hash
	return crypto.Keccak256Hash(rawData).Bytes(), nil
}

// Constructs the EIP712 formatted bid
func constructPreConfirmationPayload(
	txHash string,
	bid *big.Int,
	blockNumber *big.Int,
	bidHash []byte,
	signature []byte,
) apitypes.TypedData {
	signerData := apitypes.TypedData{
		Types: apitypes.Types{
			"PreConfPreConfirmation": []apitypes.Type{
				{Name: "txHash", Type: "string"},
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
			"txHash":      txHash,
			"bid":         bid,
			"blockNumber": blockNumber,
			"bidHash":     hex.EncodeToString(bidHash),
			"signature":   hex.EncodeToString(signature),
		},
	}

	return signerData
}

// Constructs the EIP712 formatted bid
func constructBidPayload(txHash string, bid *big.Int, blockNumber *big.Int) apitypes.TypedData {
	signerData := apitypes.TypedData{
		Types: apitypes.Types{
			"PreConfBid": []apitypes.Type{
				{Name: "txHash", Type: "string"},
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
			"txHash":      txHash,
			"bid":         bid,
			"blockNumber": blockNumber,
		},
	}

	return signerData
}
