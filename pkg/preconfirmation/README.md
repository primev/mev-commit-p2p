## Overview

The preconfirmation package creates a simple system where two types of users, referred to as users and providers, can exchange bid requests and confirmations over a peer-to-peer network. Users use the SendBid function to send bids and wait for confirmations from providers. Providers use the handleBid function to receive bids, check them, and send back confirmations if the bids are valid. 

### Diagram
![](preconf-mc.png)

## Data Structures

There are three key data structures:
- Unsigned Bid
- Signed Bid
- Signed Commitment (Pre-Confirmation)

### Unsigned Bid

```go
type UnsignedBid struct {
	TxHash      string   `json:"txn_hash"` // UUID for bundle or txn
	BidAmt      *big.Int `json:"bid_amt"` // Wei amount of bid
	BlockNumber *big.Int `json:"block_number"` // expiring blocknumber
}
```

```go
type SignedBid struct {
	TxHash      string   `json:"txn_hash"` // UUID for bundle or txn
	BidAmt      *big.Int `json:"bid_amt"` // Wei amount of bid
	BlockNumber *big.Int `json:"block_number"` // expiring blocknumber

	Digest    []byte `json:"bid_digest"` // The hash of the above payload encoded in EIP-712 format
	Signature []byte `json:"bid_signature"` // Signature of Digest, signed by the User Private Key
}
```

```go
type PreConfirmation struct {
	Bid SignedBid `json:"bid"` // Contains the Signed Bid

	Digest    []byte `json:"digest"` // Is the EIP-712 formated hash of the signed bid
	Signature []byte `json:"signature"` // Is the Digest above signed by the provider private key
}

```
