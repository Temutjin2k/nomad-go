package hasher

import (
	"crypto/sha256"
	"encoding/hex"
)

// Hash возвращает SHA-256 хэш входной строки в виде hex.
func Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func Verify(pass, hash string) bool {
	return Hash(pass) == hash
}

// SumBytes — та же функция, но на вход принимает []byte.
func SumBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
