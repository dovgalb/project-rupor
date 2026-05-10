package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func TestNewPassword(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raw     string
		wantErr error
	}{
		{"минимум 8", "12345678", nil},
		{"средняя", "qwerty123", nil},
		{"максимум 72", strings.Repeat("a", 72), nil},
		{"меньше 8", "1234", domain.ErrInvalidPassword},
		{"пусто", "", domain.ErrInvalidPassword},
		{"больше 72", strings.Repeat("a", 73), domain.ErrInvalidPassword},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, err := domain.NewPassword(tc.raw)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.String() != tc.raw {
				t.Fatalf("got %q, want %q", p.String(), tc.raw)
			}
		})
	}
}
