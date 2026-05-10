package domain_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func TestNewTokenHash_WrongLength(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  []byte
	}{
		{"пусто", nil},
		{"len=1", make([]byte, 1)},
		{"len=31", make([]byte, 31)},
		{"len=33", make([]byte, 33)},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewTokenHash(tc.raw)
			if !errors.Is(err, domain.ErrInvalidTokenHash) {
				t.Fatalf("len=%d: got %v, want ErrInvalidTokenHash", len(tc.raw), err)
			}
		})
	}

	t.Run("len=32 OK", func(t *testing.T) {
		t.Parallel()
		raw := bytes.Repeat([]byte{0xAB}, 32)
		h, err := domain.NewTokenHash(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := h.Bytes()
		if !bytes.Equal(got[:], raw) {
			t.Fatalf("Bytes() does not match input")
		}
	})
}

func TestTokenHash_Equal(t *testing.T) {
	t.Parallel()

	a := mustTokenHash(t, bytes.Repeat([]byte{0x01}, 32))
	b := mustTokenHash(t, bytes.Repeat([]byte{0x01}, 32))
	c := mustTokenHash(t, bytes.Repeat([]byte{0x02}, 32))

	if !a.Equal(b) {
		t.Fatalf("expected equal hashes")
	}
	if a.Equal(c) {
		t.Fatalf("expected different hashes")
	}
}
