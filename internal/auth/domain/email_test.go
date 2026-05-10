package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func TestNewEmail(t *testing.T) {
	t.Parallel()

	t.Run("валидные c trim+lowercase", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			raw  string
			want string
		}{
			{"user@example.com", "user@example.com"},
			{"  User@Example.COM  ", "user@example.com"},
			{"a.b+c@sub.example.org", "a.b+c@sub.example.org"},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.raw, func(t *testing.T) {
				t.Parallel()
				e, err := domain.NewEmail(tc.raw)
				if err != nil {
					t.Fatalf("NewEmail(%q) unexpected error: %v", tc.raw, err)
				}
				if e.String() != tc.want {
					t.Fatalf("got %q, want %q", e.String(), tc.want)
				}
			})
		}
	})

	t.Run("невалидные", func(t *testing.T) {
		t.Parallel()

		long := strings.Repeat("a", 250) + "@b.co" // > 254
		cases := []struct {
			name string
			raw  string
		}{
			{"пусто", ""},
			{"только пробелы", "   "},
			{"без @", "userexample.com"},
			{"без точки в домене", "user@example"},
			{"пробел внутри", "us er@example.com"},
			{"дважды @", "a@b@c.com"},
			{"длиннее 254", long},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := domain.NewEmail(tc.raw)
				if !errors.Is(err, domain.ErrInvalidEmail) {
					t.Fatalf("raw %q: got %v, want ErrInvalidEmail", tc.raw, err)
				}
			})
		}
	})

	t.Run("сравнение equal", func(t *testing.T) {
		t.Parallel()
		a := mustEmail(t, "User@Example.com")
		b := mustEmail(t, "user@example.com")
		if a != b {
			t.Fatalf("expected emails to be equal after normalization")
		}
	})
}
