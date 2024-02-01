package signer

import "github.com/ethereum/go-ethereum/accounts"

// KeyStorer is an interface for mocking KeyStore
type KeyStoreSigner interface {
	SignHashWithPassphrase(a accounts.Account, passphrase string, hash []byte) ([]byte, error)
}
