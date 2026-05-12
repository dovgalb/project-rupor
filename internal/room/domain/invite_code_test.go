package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestNewInviteCode_Valid_Normalizes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"already upper", "ABCDEFGH", "ABCDEFGH"},
		{"lower to upper", "abcdefgh", "ABCDEFGH"},
		{"with surrounding spaces", "  ABCDEFGH  ", "ABCDEFGH"},
		{"digits only", "01234567", "01234567"},
		{"mixed alphabet", "0123ABCD", "0123ABCD"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewInviteCode(tc.input)
			if err != nil {
				t.Fatalf("NewInviteCode(%q): %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("NewInviteCode(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestNewInviteCode_InvalidLength_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"7 chars", "ABCDEFG"},
		{"9 chars", "ABCDEFGHI"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewInviteCode(tc.input)
			if !errors.Is(err, domain.ErrInvalidInviteCode) {
				t.Fatalf("got %v, want ErrInvalidInviteCode", err)
			}
		})
	}
}

func TestNewInviteCode_InvalidChar_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"contains I", "ABCDIFGH"},
		{"contains L", "ABCDLFGH"},
		{"contains O", "ABCDOFGH"},
		{"contains U", "ABCDUFGH"},
		{"contains dash", "ABCD-FGH"},
		{"contains asterisk", "ABCD*FGH"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewInviteCode(tc.input)
			if !errors.Is(err, domain.ErrInvalidInviteCode) {
				t.Fatalf("got %v, want ErrInvalidInviteCode", err)
			}
		})
	}
}

func TestInviteCodeAlphabet_HasNoConfusables(t *testing.T) {
	t.Parallel()

	a := domain.InviteCodeAlphabet()
	if len(a) != 32 {
		t.Fatalf("alphabet length = %d, want 32", len(a))
	}
	for _, ch := range []string{"I", "L", "O", "U"} {
		if strings.Contains(a, ch) {
			t.Fatalf("alphabet contains confusable %q", ch)
		}
	}
}
