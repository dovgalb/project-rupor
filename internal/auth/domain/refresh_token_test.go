package domain_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func newActiveToken(t *testing.T) (*domain.RefreshToken, time.Time, time.Time) {
	t.Helper()

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	expires := now.Add(24 * time.Hour)
	tok, err := domain.NewRefreshToken(
		mustRefreshTokenID(t, uuid.New()),
		mustUserID(t, uuid.New()),
		mustTokenHash(t, bytes.Repeat([]byte{0x01}, 32)),
		expires,
		now,
	)
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	return tok, now, expires
}

func TestNewRefreshToken_Success(t *testing.T) {
	t.Parallel()

	tok, now, _ := newActiveToken(t)
	if !tok.IsActive(now) {
		t.Fatalf("expected active token")
	}
}

func TestNewRefreshToken_BadExpiration(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		expires time.Time
	}{
		{"равно createdAt", now},
		{"раньше createdAt", now.Add(-time.Hour)},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewRefreshToken(
				mustRefreshTokenID(t, uuid.New()),
				mustUserID(t, uuid.New()),
				mustTokenHash(t, bytes.Repeat([]byte{0x01}, 32)),
				tc.expires,
				now,
			)
			if !errors.Is(err, domain.ErrInvalidRefreshTokenExpiration) {
				t.Fatalf("got %v, want ErrInvalidRefreshTokenExpiration", err)
			}
		})
	}
}

func TestRefreshToken_Revoke_Success(t *testing.T) {
	t.Parallel()

	tok, now, _ := newActiveToken(t)
	revokeAt := now.Add(time.Minute)

	if err := tok.Revoke(revokeAt); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if !tok.IsRevoked() {
		t.Fatalf("expected revoked")
	}
	if !tok.RevokedAt().Equal(revokeAt) {
		t.Fatalf("RevokedAt = %s, want %s", tok.RevokedAt(), revokeAt)
	}
}

func TestRefreshToken_Revoke_Twice(t *testing.T) {
	t.Parallel()

	tok, now, _ := newActiveToken(t)
	first := now.Add(time.Minute)
	if err := tok.Revoke(first); err != nil {
		t.Fatalf("first Revoke: %v", err)
	}

	err := tok.Revoke(now.Add(2 * time.Minute))
	if !errors.Is(err, domain.ErrRefreshTokenAlreadyRevoked) {
		t.Fatalf("got %v, want ErrRefreshTokenAlreadyRevoked", err)
	}
	if !tok.RevokedAt().Equal(first) {
		t.Fatalf("RevokedAt changed after second Revoke")
	}
}

func TestRefreshToken_IsActive(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)
	afterExpiry := expires.Add(time.Second)

	cases := []struct {
		name    string
		revoked bool
		check   time.Time
		want    bool
	}{
		{"active", false, now, true},
		{"revoked", true, now, false},
		{"expired", false, afterExpiry, false},
		{"revoked + expired", true, afterExpiry, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tok, err := domain.NewRefreshToken(
				mustRefreshTokenID(t, uuid.New()),
				mustUserID(t, uuid.New()),
				mustTokenHash(t, bytes.Repeat([]byte{0x01}, 32)),
				expires,
				now,
			)
			if err != nil {
				t.Fatalf("NewRefreshToken: %v", err)
			}
			if tc.revoked {
				if err := tok.Revoke(now); err != nil {
					t.Fatalf("Revoke: %v", err)
				}
			}
			if got := tok.IsActive(tc.check); got != tc.want {
				t.Fatalf("IsActive = %v, want %v", got, tc.want)
			}
		})
	}
}
