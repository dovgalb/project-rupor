package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

func TestNewChannelName_Valid_Normalizes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"typical", "general", "general"},
		{"leading-trailing spaces", "  general  ", "general"},
		{"max length 64", strings.Repeat("a", 64), strings.Repeat("a", 64)},
		{"max length 64 multibyte", strings.Repeat("я", 64), strings.Repeat("я", 64)},
		{"min length 1", "a", "a"},
		{"unicode", "канал-1", "канал-1"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewChannelName(tc.input)
			if err != nil {
				t.Fatalf("NewChannelName(%q): %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("NewChannelName(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestNewChannelName_TooShort_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"only spaces", "    "},
		{"only tabs", "\t\t"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewChannelName(tc.input)
			if !errors.Is(err, domain.ErrInvalidChannelName) {
				t.Fatalf("got %v, want ErrInvalidChannelName", err)
			}
		})
	}
}

func TestNewChannelName_TooLong_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"65 chars", strings.Repeat("a", 65)},
		{"100 chars", strings.Repeat("a", 100)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewChannelName(tc.input)
			if !errors.Is(err, domain.ErrInvalidChannelName) {
				t.Fatalf("got %v, want ErrInvalidChannelName", err)
			}
		})
	}
}

func TestNewChannelName_ControlCharacter_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"newline", "chan\nnel"},
		{"tab", "chan\tnel"},
		{"carriage return", "chan\rnel"},
		{"null byte", "chan\x00nel"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewChannelName(tc.input)
			if !errors.Is(err, domain.ErrInvalidChannelName) {
				t.Fatalf("got %v, want ErrInvalidChannelName", err)
			}
		})
	}
}

// Format-управляющие (Cf): RLO, ZWSP, BOM, LRM — защита от UI-spoofing.
// Записаны через \u-escape для ASCII-safe исходника.
func TestNewChannelName_FormatCharacter_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"rtl override (U+202E)", "\u202Echannel"},
		{"zero width space (U+200B)", "chan\u200Bnel"},
		{"byte order mark (U+FEFF)", "channel\uFEFF"},
		{"left-to-right mark (U+200E)", "channel\u200E"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewChannelName(tc.input)
			if !errors.Is(err, domain.ErrInvalidChannelName) {
				t.Fatalf("got %v, want ErrInvalidChannelName", err)
			}
		})
	}
}
