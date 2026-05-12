package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestNewRoomName_Valid_Normalizes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"typical", "general", "general"},
		{"leading-trailing spaces", "  команда  ", "команда"},
		{"max length (64 runes)", strings.Repeat("a", 64), strings.Repeat("a", 64)},
		{"max length 64 runes multibyte", strings.Repeat("я", 64), strings.Repeat("я", 64)},
		{"min length (1 rune)", "a", "a"},
		{"unicode name", "комната № 1", "комната № 1"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewRoomName(tc.input)
			if err != nil {
				t.Fatalf("NewRoomName(%q): %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("NewRoomName(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestNewRoomName_TooShort_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"only spaces", "    "},
		{"only tabs and spaces", "\t  \t"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewRoomName(tc.input)
			if !errors.Is(err, domain.ErrInvalidRoomName) {
				t.Fatalf("got %v, want ErrInvalidRoomName", err)
			}
		})
	}
}

func TestNewRoomName_TooLong_ReturnsErr(t *testing.T) {
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
			_, err := domain.NewRoomName(tc.input)
			if !errors.Is(err, domain.ErrInvalidRoomName) {
				t.Fatalf("got %v, want ErrInvalidRoomName", err)
			}
		})
	}
}

func TestNewRoomName_ControlCharacter_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"newline", "room\nname"},
		{"tab", "room\tname"},
		{"carriage return", "room\rname"},
		{"null byte", "room\x00name"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewRoomName(tc.input)
			if !errors.Is(err, domain.ErrInvalidRoomName) {
				t.Fatalf("got %v, want ErrInvalidRoomName", err)
			}
		})
	}
}

// TestNewRoomName_FormatCharacter_ReturnsErr — Cf-категория (формат-управляющие).
// Защита от UI-spoofing (RLO, LRM) и невидимых дублей имён (ZWSP, BOM).
// Управляющие коды записаны через \u-escapes, чтобы исходник был ASCII-safe
// (BOM-литералы в начале строкового литерала запрещены компилятором).
func TestNewRoomName_FormatCharacter_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"rtl override (U+202E)", "\u202EАдмин"},
		{"zero width space (U+200B)", "general\u200B"},
		{"byte order mark (U+FEFF)", "room\uFEFF"},
		{"left-to-right mark (U+200E)", "room\u200E"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewRoomName(tc.input)
			if !errors.Is(err, domain.ErrInvalidRoomName) {
				t.Fatalf("got %v, want ErrInvalidRoomName", err)
			}
		})
	}
}
