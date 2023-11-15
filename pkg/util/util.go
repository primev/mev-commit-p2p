package util

import (
	"io"
	"log/slog"
	"math/big"
)

func PadKeyTo32Bytes(key *big.Int) []byte {
	keyBytes := key.Bytes()
	if len(keyBytes) < 32 {
		padding := make([]byte, 32-len(keyBytes))
		keyBytes = append(padding, keyBytes...)
	}
	return keyBytes
}

func NewTestLogger(w io.Writer) *slog.Logger {
	testLogger := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return slog.New(testLogger)
}
