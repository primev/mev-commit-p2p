package util

import (
	"crypto/ecdsa"
	"io"
	"log/slog"
	"math/big"
	"runtime"
)

func PadKeyTo32Bytes(key *big.Int) []byte {
	keyBytes := key.Bytes()
	if len(keyBytes) < 32 {
		padding := make([]byte, 32-len(keyBytes))
		keyBytes = append(padding, keyBytes...)
	}
	return keyBytes
}

func ZeroPrivateKey(key *ecdsa.PrivateKey) {
	b := key.D.Bits()
	for i := range b {
		b[i] = 0
	}
	// Force garbage collection to remove the key from memory
	runtime.GC()
}

func NewTestLogger(w io.Writer) *slog.Logger {
	testLogger := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return slog.New(testLogger)
}
