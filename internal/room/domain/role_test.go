package domain_test

import (
	"errors"
	"testing"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestParseRole_TableDriven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    domain.Role
		wantErr error
	}{
		{"owner", "owner", domain.RoleOwner, nil},
		{"admin", "admin", domain.RoleAdmin, nil},
		{"member", "member", domain.RoleMember, nil},
		{"empty", "", 0, domain.ErrInvalidRole},
		{"upper-case", "OWNER", 0, domain.ErrInvalidRole},
		{"foo", "foo", 0, domain.ErrInvalidRole},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseRole(tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ParseRole(%q) err = %v, want %v", tc.input, err, tc.wantErr)
			}
			if err == nil && got != tc.want {
				t.Fatalf("ParseRole(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestRole_String_TableDriven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		role domain.Role
		want string
	}{
		{"owner", domain.RoleOwner, "owner"},
		{"admin", domain.RoleAdmin, "admin"},
		{"member", domain.RoleMember, "member"},
		{"invalid", domain.Role(42), "invalid"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.role.String(); got != tc.want {
				t.Fatalf("Role(%d).String() = %q, want %q", tc.role, got, tc.want)
			}
		})
	}
}

func TestRole_RoundTrip_ParseStringParse(t *testing.T) {
	t.Parallel()

	roles := []domain.Role{domain.RoleMember, domain.RoleAdmin, domain.RoleOwner}
	for _, r := range roles {
		r := r
		t.Run(r.String(), func(t *testing.T) {
			t.Parallel()
			parsed, err := domain.ParseRole(r.String())
			if err != nil {
				t.Fatalf("ParseRole(%q): %v", r.String(), err)
			}
			if parsed != r {
				t.Fatalf("round-trip: got %v, want %v", parsed, r)
			}
		})
	}
}

func TestRole_IsValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		role domain.Role
		want bool
	}{
		{"member", domain.RoleMember, true},
		{"admin", domain.RoleAdmin, true},
		{"owner", domain.RoleOwner, true},
		{"unknown", domain.Role(99), false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.role.IsValid(); got != tc.want {
				t.Fatalf("Role(%d).IsValid() = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}
