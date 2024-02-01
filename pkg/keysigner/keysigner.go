package keysigner

import (
	"crypto/ecdsa"
	"math/big"
	"os"
	"runtime"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type KeySigner interface {
	SignHash(data []byte) ([]byte, error)
	SignTx(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
	GetAddress() common.Address
	GetPrivateKey() (*ecdsa.PrivateKey, error)
}

type privateKeySigner struct {
	privKey *ecdsa.PrivateKey
}

func NewPrivateKeySigner(privKey *ecdsa.PrivateKey) *privateKeySigner {
	return &privateKeySigner{
		privKey: privKey,
	}
}

func (pks *privateKeySigner) SignHash(hash []byte) ([]byte, error) {
	return crypto.Sign(hash, pks.privKey)
}

func (pks *privateKeySigner) SignTx(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return types.SignTx(tx, types.NewLondonSigner(chainID), pks.privKey)
}

func (pks *privateKeySigner) GetAddress() common.Address {
	return crypto.PubkeyToAddress(pks.privKey.PublicKey)
}

func (pks *privateKeySigner) GetPrivateKey() (*ecdsa.PrivateKey, error) {
	return pks.privKey, nil
}

type keystoreSigner struct {
	keystore *keystore.KeyStore
	password string
	account  accounts.Account
}

func NewKeystoreSigner(keystore *keystore.KeyStore, password string, account accounts.Account) *keystoreSigner {
	return &keystoreSigner{
		keystore: keystore,
		password: password,
		account:  account,
	}
}

func (kss *keystoreSigner) SignHash(hash []byte) ([]byte, error) {
	return kss.keystore.SignHashWithPassphrase(kss.account, kss.password, hash)
}

func (kss *keystoreSigner) SignTx(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return kss.keystore.SignTxWithPassphrase(kss.account, kss.password, tx, chainID)
}

func (kss *keystoreSigner) GetAddress() common.Address {
	return kss.account.Address
}

func (kss *keystoreSigner) GetPrivateKey() (*ecdsa.PrivateKey, error) {
	return extractPrivateKey(kss.account.URL.Path, kss.password)
}

func extractPrivateKey(keystoreFile, passphrase string) (*ecdsa.PrivateKey, error) {
	keyjson, err := os.ReadFile(keystoreFile)
	if err != nil {
		return nil, err
	}

	key, err := keystore.DecryptKey(keyjson, passphrase)
	if err != nil {
		return nil, err
	}

	// Zero out the keyjson slice
	for i := range keyjson {
		keyjson[i] = 0
	}

	return key.PrivateKey, nil
}

func ZeroPrivateKey(key *ecdsa.PrivateKey) {
	b := key.D.Bits()
	for i := range b {
		b[i] = 0
	}
	// Force garbage collection to remove the key from memory
	runtime.GC()
}
