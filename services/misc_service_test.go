package services

import (
	"net"
	"strings"
	"testing"
)

func TestFreePort(t *testing.T) {
	p, err := FreePort()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if p <= 0 || p > 65535 {
		t.Errorf("invalid port: %d", p)
	}
	// Port should actually be bindable.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	_ = l.Close()
}

func TestNewUUID(t *testing.T) {
	full := NewUUID(false)
	if len(full) != 36 {
		t.Errorf("expected 36-char UUID, got %d: %s", len(full), full)
	}
	short := NewUUID(true)
	if len(short) != 8 {
		t.Errorf("expected 8-char short UUID, got %d: %s", len(short), short)
	}
}

func TestRandomString(t *testing.T) {
	s, err := RandomString(32, "hex")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(s) != 32 {
		t.Errorf("expected 32 chars, got %d", len(s))
	}
	for _, ch := range s {
		if !strings.ContainsRune("0123456789abcdef", ch) {
			t.Errorf("non-hex char %q in %q", ch, s)
		}
	}
}

func TestRandomString_UnknownAlphabet(t *testing.T) {
	if _, err := RandomString(8, "klingon"); err == nil {
		t.Error("expected error for unknown alphabet")
	}
}

func TestRandomString_AlphabetsValid(t *testing.T) {
	for name := range AlphabetMap {
		s, err := RandomString(10, name)
		if err != nil {
			t.Errorf("alphabet %s: %v", name, err)
			continue
		}
		if len(s) != 10 {
			t.Errorf("alphabet %s: bad length", name)
		}
	}
}
