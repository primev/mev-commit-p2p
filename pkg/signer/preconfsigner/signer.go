package preconfsigner

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/primevprotocol/mev-commit/pkg/keysigner"
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

	// The format for these timestamps is unix timestamp in milliseconds
	DecayStartTimeStamp uint64 `json:"decay_start_timestamp"`
	DecayEndTimeStamp   uint64 `json:"decay_end_timestamp"`

	Digest    []byte `json:"bid_digest"` // TODO(@ckaritk): name better
	Signature []byte `json:"bid_signature"`
}

func (b Bid) String() string {
	return fmt.Sprintf(
		"TxHash: %s, BidAmt: %s, BlockNumber: %s, Digest: %s, Signature: %s, DecayStartTimeStamp: %d, DecayEndTimeStamp: %d",
		b.TxHash, b.BidAmt, b.BlockNumber, hex.EncodeToString(b.Digest), hex.EncodeToString(b.Signature), b.DecayStartTimeStamp, b.DecayEndTimeStamp,
	)
}

type PreConfirmation struct {
	Bid Bid `json:"bid"`

	Digest    []byte `json:"digest"` // TODO(@ckaritk): name better
	Signature []byte `json:"signature"`

	ProviderAddress common.Address `json:"provider_address"`
}

func (p PreConfirmation) String() string {
	return fmt.Sprintf(
		"Bid: %s, Digest: %s, Signature: %s",
		p.Bid, hex.EncodeToString(p.Digest), hex.EncodeToString(p.Signature),
	)
}

type Signer interface {
	ConstructSignedBid(string, *big.Int, *big.Int, uint64, uint64) (*Bid, error)
	ConstructPreConfirmation(*Bid) (*PreConfirmation, error)
	VerifyBid(*Bid) (*common.Address, error)
	VerifyPreConfirmation(*PreConfirmation) (*common.Address, error)
}

type privateKeySigner struct {
	keySigner keysigner.KeySigner
}

func NewSigner(keySigner keysigner.KeySigner) *privateKeySigner {
	return &privateKeySigner{
		keySigner: keySigner,
	}
}

func (p *privateKeySigner) ConstructSignedBid(
	txHash string,
	bidAmt *big.Int,
	blockNumber *big.Int,
	decayStartTimeStamp uint64,
	decayEndTimeStamp uint64,
) (*Bid, error) {
	if txHash == "" || bidAmt == nil || blockNumber == nil {
		return nil, errors.New("missing required fields")
	}

	bid := &Bid{
		BidAmt:              bidAmt,
		TxHash:              txHash,
		BlockNumber:         blockNumber,
		DecayStartTimeStamp: decayStartTimeStamp,
		DecayEndTimeStamp:   decayEndTimeStamp,
	}

	bidHash, err := GetBidHash(bid)
	if err != nil {
		return nil, err
	}

	sig, err := p.keySigner.SignHash(bidHash)
	if err != nil {
		return nil, err
	}

	if sig[64] == 0 || sig[64] == 1 {
		sig[64] += 27 // Transform V from 0/1 to 27/28
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

	sig, err := p.keySigner.SignHash(preConfirmationHash)
	if err != nil {
		return nil, err
	}

	if sig[64] == 0 || sig[64] == 1 {
		sig[64] += 27 // Transform V from 0/1 to 27/28
	}

	preConfirmation.Digest = preConfirmationHash
	preConfirmation.Signature = sig

	return preConfirmation, nil
}

func (p *privateKeySigner) VerifyBid(bid *Bid) (*common.Address, error) {
	if bid.Digest == nil || bid.Signature == nil {
		return nil, ErrMissingHashSignature
	}

	bidHash, err := GetBidHash(bid)
	if err != nil {
		return nil, err
	}

	return eipVerify(
		bidHash,
		bid.Digest,
		bid.Signature,
	)
}

// VerifyPreConfirmation verifies the preconfirmation message, and returns the address of the provider
// that signed the preconfirmation.
func (p *privateKeySigner) VerifyPreConfirmation(c *PreConfirmation) (*common.Address, error) {
	if c.Digest == nil || c.Signature == nil {
		return nil, ErrMissingHashSignature
	}

	_, err := p.VerifyBid(&c.Bid)
	if err != nil {
		return nil, err
	}

	preConfirmationHash, err := GetPreConfirmationHash(c)
	if err != nil {
		return nil, err
	}

	return eipVerify(preConfirmationHash, c.Digest, c.Signature)
}

func eipVerify(
	payloadHash []byte,
	expectedhash []byte,
	signature []byte,
) (*common.Address, error) {
	if !bytes.Equal(payloadHash, expectedhash) {
		return nil, ErrInvalidHash
	}

	sig := make([]byte, len(signature))
	copy(sig, signature)
	if sig[64] >= 27 && sig[64] <= 28 {
		sig[64] -= 27
	}

	pubkey, err := crypto.SigToPub(payloadHash, sig)
	if err != nil {
		return nil, err
	}

	if !crypto.VerifySignature(
		crypto.FromECDSAPub(pubkey),
		payloadHash,
		sig[:len(sig)-1],
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
		[]byte("PreConfBid(string txnHash,uint64 bid,uint64 blockNumber,uint64 decayStartTimeStamp,uint64 decayEndTimeStamp)"),
	)

	// Convert the txnHash to a byte array and hash it
	txnHashHash := crypto.Keccak256Hash([]byte(bid.TxHash))

	// Encode values similar to Solidity's abi.encode
	// The reason we use math.U256Bytes is because we want to encode the uint64 as a 32 byte array
	// The EVM does this for values due via padding to 32 bytes, as that's the base size of a word in the EVM
	data := append(eip712MessageTypeHash.Bytes(), txnHashHash.Bytes()...)
	data = append(data, math.U256Bytes(bid.BidAmt)...)
	data = append(data, math.U256Bytes(bid.BlockNumber)...)
	data = append(data, math.U256Bytes(big.NewInt(int64(bid.DecayStartTimeStamp)))...)
	data = append(data, math.U256Bytes(big.NewInt(int64(bid.DecayEndTimeStamp)))...)
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
		[]byte("PreConfCommitment(string txnHash,uint64 bid,uint64 blockNumber,uint64 decayStartTimeStamp,uint64 decayEndTimeStamp,string bidHash,string signature)"),
	)

	// Convert the txnHash to a byte array and hash it
	txnHashHash := crypto.Keccak256Hash([]byte(c.Bid.TxHash))
	bidDigestHash := crypto.Keccak256Hash([]byte(hex.EncodeToString(c.Bid.Digest)))
	bidSigHash := crypto.Keccak256Hash([]byte(hex.EncodeToString(c.Bid.Signature)))

	// Encode values similar to Solidity's abi.encode
	data := append(eip712MessageTypeHash.Bytes(), txnHashHash.Bytes()...)
	data = append(data, math.U256Bytes(c.Bid.BidAmt)...)
	data = append(data, math.U256Bytes(c.Bid.BlockNumber)...)
	data = append(data, math.U256Bytes(new(big.Int).SetUint64(c.Bid.DecayStartTimeStamp))...)
	data = append(data, math.U256Bytes(big.NewInt(int64(c.Bid.DecayEndTimeStamp)))...)
	data = append(data, bidDigestHash.Bytes()...)
	data = append(data, bidSigHash.Bytes()...)
	dataHash := crypto.Keccak256Hash(data)

	rawData := append([]byte("\x19\x01"), append(domainSeparatorBid.Bytes(), dataHash.Bytes()...)...)
	// Create the final hash
	return crypto.Keccak256Hash(rawData).Bytes(), nil
}

// Constructs the EIP712 formatted bid
// nolint:unused
func constructPreConfirmationPayload(
	txHash string,
	bid *big.Int,
	blockNumber *big.Int,
	decayStartTimeStamp *big.Int,
	decayEndTimeStamp *big.Int,
	bidHash []byte,
	signature []byte,
) apitypes.TypedData {
	signerData := apitypes.TypedData{
		Types: apitypes.Types{
			"PreConfPreConfirmation": []apitypes.Type{
				{Name: "txHash", Type: "string"},
				{Name: "bid", Type: "uint64"},
				{Name: "blockNumber", Type: "uint64"},
				{Name: "decayStartTimeStamp", Type: "uint64"},
				{Name: "decayEndTimeStamp", Type: "uint64"},
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
			"txHash":              txHash,
			"bid":                 bid,
			"blockNumber":         blockNumber,
			"decayStartTimeStamp": decayStartTimeStamp,
			"decayEndTimeStamp":   decayEndTimeStamp,
			"bidHash":             hex.EncodeToString(bidHash),
			"signature":           hex.EncodeToString(signature),
		},
	}

	return signerData
}

// Constructs the EIP712 formatted bid
// nolint:unused
func constructBidPayload(txHash string, bid *big.Int, blockNumber *big.Int, decayStartTimeStamp *big.Int, decayEndTimeStamp *big.Int) apitypes.TypedData {
	signerData := apitypes.TypedData{
		Types: apitypes.Types{
			"PreConfBid": []apitypes.Type{
				{Name: "txHash", Type: "string"},
				{Name: "bid", Type: "uint64"},
				{Name: "blockNumber", Type: "uint64"},
				{Name: "decayStartTimeStamp", Type: "uint64"},
				{Name: "decayEndTimeStamp", Type: "uint64"},
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
			"txHash":              txHash,
			"bid":                 bid,
			"blockNumber":         blockNumber,
			"decayStartTimeStamp": decayStartTimeStamp,
			"decayEndTimeStamp":   decayEndTimeStamp,
		},
	}

	return signerData
}
