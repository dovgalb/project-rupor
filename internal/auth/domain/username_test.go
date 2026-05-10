package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func TestNewUsername(t *testing.T) {
	t.Parallel()

	t.Run("валидные", func(t *testing.T) {
		t.Parallel()
		cases := []string{"abc", "user_1", "User-2", strings.Repeat("a", 32)}
		for _, raw := range cases {
			raw := raw
			t.Run(raw, func(t *testing.T) {
				t.Parallel()
				u, err := domain.NewUsername(raw)
				if err != nil {
					t.Fatalf("NewUsername(%q): %v", raw, err)
				}
				if u.String() != strings.TrimSpace(raw) {
					t.Fatalf("got %q, want %q", u.String(), raw)
				}
			})
		}
	})

	t.Run("невалидные", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			raw  string
		}{
			{"короткий", "ab"},
			{"длинный", strings.Repeat("a", 33)},
			{"пробел внутри", "ab cd"},
			{"плюс", "user+1"},
			{"@", "user@1"},
			{"кириллица", "пользователь"},
			{"пусто", ""},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := domain.NewUsername(tc.raw)
				if !errors.Is(err, domain.ErrInvalidUsername) {
					t.Fatalf("raw %q: got %v, want ErrInvalidUsername", tc.raw, err)
				}
			})
		}
	})
}
