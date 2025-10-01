package uuid

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// UUID — 16 байт
type UUID [16]byte

// New возвращает новый UUID v4
func New() (UUID, error) {
	var u UUID
	if _, err := io.ReadFull(rand.Reader, u[:]); err != nil {
		return UUID{}, err
	}
	// версия (4)
	u[6] = (u[6] & 0x0f) | 0x40
	// вариант (RFC 4122)
	u[8] = (u[8] & 0x3f) | 0x80
	return u, nil
}

// String форматирует UUID в XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
func (u UUID) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

// Parse парсит строку в UUID
func Parse(s string) (UUID, error) {
	var u UUID
	s = strings.ToLower(s)

	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return UUID{}, errors.New("invalid uuid format")
	}
	joined := strings.Join(parts, "")
	if len(joined) != 32 {
		return UUID{}, errors.New("invalid uuid length")
	}

	b, err := hex.DecodeString(joined)
	if err != nil {
		return UUID{}, err
	}
	copy(u[:], b)
	return u, nil
}

// ParseBytes парсит UUID из []byte
func ParseBytes(b []byte) (UUID, error) {
	return Parse(string(b))
}

// --- JSON ---
// MarshalJSON сериализует UUID в строку
func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

// UnmarshalJSON парсит UUID из JSON строки
func (u *UUID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

// --- Text ---
// MarshalText implements encoding.TextMarshaler.
func (u UUID) MarshalText() ([]byte, error) {
	var js [36]byte
	encodeHex(js[:], u)
	return js[:], nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (u *UUID) UnmarshalText(data []byte) error {
	id, err := ParseBytes(data)
	if err != nil {
		return err
	}
	*u = id
	return nil
}

// --- Binary ---
// MarshalBinary implements encoding.BinaryMarshaler.
func (u UUID) MarshalBinary() ([]byte, error) {
	return u[:], nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (u *UUID) UnmarshalBinary(data []byte) error {
	if len(data) != 16 {
		return fmt.Errorf("invalid UUID (got %d bytes)", len(data))
	}
	copy(u[:], data)
	return nil
}

// --- Helpers ---
// encodeHex форматирует UUID в canonical текст (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
func encodeHex(dst []byte, u UUID) {
	hex.Encode(dst[0:8], u[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], u[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], u[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], u[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], u[10:16])
}
