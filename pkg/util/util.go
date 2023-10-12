package util

import "math/big"

func PadKeyTo32Bytes(key *big.Int) []byte {
	keyBytes := key.Bytes()
	if len(keyBytes) < 32 {
		padding := make([]byte, 32-len(keyBytes))
		keyBytes = append(padding, keyBytes...)
	}
	return keyBytes
}
