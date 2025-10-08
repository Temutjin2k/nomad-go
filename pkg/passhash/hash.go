package passhash

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Tunables — adjust to your performance budget.
// Aim for ~150–300 ms per hash on your production hardware.
const (
	DefaultIterations = 210_000
	SaltLen           = 16
	KeyLen            = 32
)

// HashPassword creates a salted PBKDF2-HMAC-SHA256 hash, returning an encoded string:
// pbkdf2_sha256$<iterations>$<saltB64>$<dkB64>
func HashPassword(password string) (string, error) {
	return HashPasswordWithIters(password, DefaultIterations)
}

func HashPasswordWithIters(password string, iterations int) (string, error) {
	if iterations <= 0 {
		return "", errors.New("iterations must be > 0")
	}
	salt := make([]byte, SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	dk := pbkdf2HMACSHA256([]byte(password), salt, iterations, KeyLen)

	enc := fmt.Sprintf(
		"pbkdf2_sha256$%d$%s$%s",
		iterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(dk),
	)
	// Best-effort wipe dk
	for i := range dk {
		dk[i] = 0
	}
	return enc, nil
}

// VerifyPassword compares a plaintext password with an encoded PBKDF2 hash.
// Returns true on match (constant-time).
func VerifyPassword(password, encoded string) (bool, error) {
	const prefix = "pbkdf2_sha256$"
	if !strings.HasPrefix(encoded, prefix) {
		return false, errors.New("unsupported hash format/prefix")
	}
	parts := strings.Split(encoded[len(prefix):], "$")
	if len(parts) != 3 {
		return false, errors.New("malformed hash")
	}

	// Parse iterations
	iters, err := strconv.Atoi(parts[0])
	if err != nil || iters <= 0 {
		return false, errors.New("invalid iterations")
	}

	// Decode salt and expected derived key
	salt, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil || len(salt) == 0 {
		return false, errors.New("invalid salt")
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(want) == 0 {
		return false, errors.New("invalid derived key")
	}

	// Recompute dk
	got := pbkdf2HMACSHA256([]byte(password), salt, iters, len(want))
	ok := subtle.ConstantTimeCompare(got, want) == 1

	// Best-effort wipe got
	for i := range got {
		got[i] = 0
	}
	return ok, nil
}

// pbkdf2HMACSHA256 implements PBKDF2 per RFC 8018 using HMAC-SHA256.
// Only stdlib used.
func pbkdf2HMACSHA256(password, salt []byte, iter, keyLen int) []byte {
	if iter <= 0 || keyLen <= 0 {
		return nil
	}
	hLen := sha256.Size
	numBlocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, numBlocks*hLen)

	var blockIndex [4]byte
	for i := 1; i <= numBlocks; i++ {
		// INT(i) big-endian
		blockIndex[0] = byte(i >> 24)
		blockIndex[1] = byte(i >> 16)
		blockIndex[2] = byte(i >> 8)
		blockIndex[3] = byte(i)

		u := hmacSHA256(password, append(salt, blockIndex[:]...))
		t := make([]byte, len(u))
		copy(t, u)

		for j := 1; j < iter; j++ {
			u = hmacSHA256(password, u)
			for k := 0; k < len(t); k++ {
				t[k] ^= u[k]
			}
		}
		out = append(out, t...)
		// best-effort wipe
		for k := range t {
			t[k] = 0
		}
	}
	return out[:keyLen]
}

func hmacSHA256(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	_, _ = m.Write(data)
	return m.Sum(nil)
}
