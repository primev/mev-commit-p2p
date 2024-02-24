package keysigner

import (
	"crypto/ecdsa"
	"fmt"
	"io"
	"math/big"
	"runtime"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type keystoreSigner struct {
	keystore *keystore.KeyStore
	password string
	account  accounts.Account
}

func NewKeystoreSigner(wr io.Writer, path, password string) (*keystoreSigner, error) {
	// lightscripts are using 4MB memory and taking approximately 100ms CPU time on a modern processor to decrypt
	keystore := keystore.NewKeyStore(path, keystore.LightScryptN, keystore.LightScryptP)
	ksAccounts := keystore.Accounts()

	var account accounts.Account
	if len(ksAccounts) == 0 {
		var err error
		account, err = keystore.NewAccount(password)
		if err != nil {
			return nil, fmt.Errorf("failed to create account: %w", err)
		}
	} else {
		account = ksAccounts[0]
	}

	fmt.Fprintf(wr, "Public address of the key: %s\n", account.Address.Hex())
	fmt.Fprintf(wr, "Path of the secret key file: %s\n", account.URL.Path)

	return &keystoreSigner{
		keystore: keystore,
		password: password,
		account:  account,
	}, nil
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

func (kss *keystoreSigner) ZeroPrivateKey(key *ecdsa.PrivateKey) {
	b := key.D.Bits()
	for i := range b {
		b[i] = 0
	}
	// Force garbage collection to remove the key from memory
	runtime.GC()
}
