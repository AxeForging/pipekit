package services

import (
	"crypto/rand"
	"fmt"
	"net"

	"github.com/google/uuid"
)

// FreePort returns an unused TCP port chosen by the OS.
func FreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// FreePortInRange returns the first unused TCP port in [low, high]. Returns
// an error if no port in the range is available.
func FreePortInRange(low, high int) (int, error) {
	if low <= 0 || high < low {
		return 0, fmt.Errorf("invalid range %d-%d", low, high)
	}
	for p := low; p <= high; p++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			_ = l.Close()
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free port in range %d-%d", low, high)
}

// NewUUID returns a v4 UUID as a string. Short=true returns the first 8 chars.
func NewUUID(short bool) string {
	u := uuid.New().String()
	if short {
		return u[:8]
	}
	return u
}

// AlphabetMap maps friendly names to character sets used by RandomString.
var AlphabetMap = map[string]string{
	"alnum":  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
	"alpha":  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
	"hex":    "0123456789abcdef",
	"base32": "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567",
	"digits": "0123456789",
	"lower":  "abcdefghijklmnopqrstuvwxyz",
	"upper":  "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
}

// RandomString returns a cryptographically random string of the given length
// drawn from the named alphabet (see AlphabetMap).
func RandomString(length int, alphabet string) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}
	chars, ok := AlphabetMap[alphabet]
	if !ok {
		return "", fmt.Errorf("unknown alphabet %q (valid: alnum, alpha, hex, base32, digits, lower, upper)", alphabet)
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for i, b := range buf {
		out[i] = chars[int(b)%len(chars)]
	}
	return string(out), nil
}
