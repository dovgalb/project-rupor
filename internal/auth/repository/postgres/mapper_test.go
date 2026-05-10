package postgres

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres/db"
)

func mustUserID(t *testing.T, raw uuid.UUID) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("NewUserID: %v", err)
	}
	return id
}

func mustRefreshTokenID(t *testing.T, raw uuid.UUID) domain.RefreshTokenID {
	t.Helper()
	id, err := domain.NewRefreshTokenID(raw)
	if err != nil {
		t.Fatalf("NewRefreshTokenID: %v", err)
	}
	return id
}

func mustEmail(t *testing.T, raw string) domain.Email {
	t.Helper()
	v, err := domain.NewEmail(raw)
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	return v
}

func mustUsername(t *testing.T, raw string) domain.Username {
	t.Helper()
	v, err := domain.NewUsername(raw)
	if err != nil {
		t.Fatalf("NewUsername: %v", err)
	}
	return v
}

func mustPasswordHash(t *testing.T, raw string) domain.PasswordHash {
	t.Helper()
	v, err := domain.NewPasswordHash(raw)
	if err != nil {
		t.Fatalf("NewPasswordHash: %v", err)
	}
	return v
}

func mustTokenHash(t *testing.T, raw []byte) domain.TokenHash {
	t.Helper()
	v, err := domain.NewTokenHash(raw)
	if err != nil {
		t.Fatalf("NewTokenHash: %v", err)
	}
	return v
}

func TestUserMapping_RoundTrip_AllFieldsPreserved(t *testing.T) {
	t.Parallel()

	originalID := uuid.New()
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	src, err := domain.NewUser(
		mustUserID(t, originalID),
		mustEmail(t, "user@example.com"),
		mustUsername(t, "user_1"),
		mustPasswordHash(t, "$2a$10$hash"),
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}

	params := domainToInsertUserParams(src)
	row := db.User(params)

	got, err := userRowToDomain(row)
	if err != nil {
		t.Fatalf("userRowToDomain: %v", err)
	}

	if got.ID() != src.ID() {
		t.Fatalf("ID mismatch")
	}
	if got.Email() != src.Email() {
		t.Fatalf("Email mismatch")
	}
	if got.Username() != src.Username() {
		t.Fatalf("Username mismatch")
	}
	if got.PasswordHash() != src.PasswordHash() {
		t.Fatalf("PasswordHash mismatch")
	}
	if !got.CreatedAt().Equal(src.CreatedAt()) {
		t.Fatalf("CreatedAt mismatch: got %s, want %s", got.CreatedAt(), src.CreatedAt())
	}
}

func TestRefreshTokenMapping_RoundTrip_AllFieldsPreserved(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	expires := now.Add(24 * time.Hour)
	revokedAt := now.Add(time.Minute)
	hashRaw := bytes.Repeat([]byte{0xAB}, 32)

	cases := []struct {
		name    string
		revoked time.Time
		valid   bool
	}{
		{"активный (revoked_at = zero, NULL в БД)", time.Time{}, false},
		{"отозванный (revoked_at != zero)", revokedAt, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src, err := domain.NewRefreshToken(
				mustRefreshTokenID(t, uuid.New()),
				mustUserID(t, uuid.New()),
				mustTokenHash(t, hashRaw),
				expires,
				now,
			)
			if err != nil {
				t.Fatalf("NewRefreshToken: %v", err)
			}
			if tc.valid {
				if revokeErr := src.Revoke(tc.revoked); revokeErr != nil {
					t.Fatalf("Revoke: %v", revokeErr)
				}
			}

			params := domainToInsertRefreshTokenParams(src)

			row := db.RefreshToken{
				ID:        params.ID,
				UserID:    params.UserID,
				TokenHash: params.TokenHash,
				ExpiresAt: params.ExpiresAt,
				CreatedAt: params.CreatedAt,
				RevokedAt: pgtype.Timestamptz{Time: tc.revoked, Valid: tc.valid},
			}

			got, err := refreshTokenRowToDomain(row)
			if err != nil {
				t.Fatalf("refreshTokenRowToDomain: %v", err)
			}

			if got.ID() != src.ID() {
				t.Fatalf("ID mismatch")
			}
			if got.UserID() != src.UserID() {
				t.Fatalf("UserID mismatch")
			}
			if !got.TokenHash().Equal(src.TokenHash()) {
				t.Fatalf("TokenHash mismatch")
			}
			if !got.ExpiresAt().Equal(src.ExpiresAt()) {
				t.Fatalf("ExpiresAt mismatch")
			}
			if !got.CreatedAt().Equal(src.CreatedAt()) {
				t.Fatalf("CreatedAt mismatch")
			}
			if !got.RevokedAt().Equal(src.RevokedAt()) {
				t.Fatalf("RevokedAt mismatch: got %s, want %s", got.RevokedAt(), src.RevokedAt())
			}
			if got.IsRevoked() != src.IsRevoked() {
				t.Fatalf("IsRevoked mismatch")
			}
		})
	}
}
