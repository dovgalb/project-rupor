package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
)

func TestNewMessageText_Valid_Normalizes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"простой", "hello", "hello"},
		{"триммит пробелы", "  hello  ", "hello"},
		{"минимум 1 руна", "a", "a"},
		{"граница 4000 рун ASCII", strings.Repeat("a", domain.MessageTextMaxLen), strings.Repeat("a", domain.MessageTextMaxLen)},
		{"граница 4000 рун multibyte", strings.Repeat("я", domain.MessageTextMaxLen), strings.Repeat("я", domain.MessageTextMaxLen)},
		{"newline разрешён", "hello\nworld", "hello\nworld"},
		{"tab разрешён", "hello\tworld", "hello\tworld"},
		{"carriage return разрешён", "hello\rworld", "hello\rworld"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewMessageText(tc.input)
			if err != nil {
				t.Fatalf("NewMessageText(%q): %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("String() = %q, want %q", got.String(), tc.want)
			}
		})
	}
}

func TestNewMessageText_Invalid_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"пустая строка", ""},
		{"только пробелы", "   "},
		{"только табы", "\t\t"},
		{"4001 руна ASCII", strings.Repeat("a", domain.MessageTextMaxLen+1)},
		{"4001 руна multibyte", strings.Repeat("я", domain.MessageTextMaxLen+1)},
		{"null byte", "hello\x00world"},
		{"bell character", "hello\aworld"},
		{"vertical tab", "hello\vworld"},
		{"zero width space (U+200B)", "hello\u200Bworld"},
		{"rtl override (U+202E)", "\u202Ehello"},
		{"byte order mark (U+FEFF)", "hello\uFEFF"},
		{"left-to-right mark (U+200E)", "hello\u200E"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewMessageText(tc.input)
			if !errors.Is(err, domain.ErrInvalidMessageText) {
				t.Fatalf("got %v, want ErrInvalidMessageText", err)
			}
		})
	}
}
