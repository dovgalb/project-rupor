package domain_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func mustEmail(t *testing.T, raw string) domain.Email {
	t.Helper()
	e, err := domain.NewEmail(raw)
	if err != nil {
		t.Fatalf("mustEmail(%q): %v", raw, err)
	}
	return e
}

func mustUsername(t *testing.T, raw string) domain.Username {
	t.Helper()
	u, err := domain.NewUsername(raw)
	if err != nil {
		t.Fatalf("mustUsername(%q): %v", raw, err)
	}
	return u
}

func mustPasswordHash(t *testing.T, raw string) domain.PasswordHash {
	t.Helper()
	h, err := domain.NewPasswordHash(raw)
	if err != nil {
		t.Fatalf("mustPasswordHash(%q): %v", raw, err)
	}
	return h
}

func mustTokenHash(t *testing.T, raw []byte) domain.TokenHash {
	t.Helper()
	h, err := domain.NewTokenHash(raw)
	if err != nil {
		t.Fatalf("mustTokenHash(len=%d): %v", len(raw), err)
	}
	return h
}

func mustUserID(t *testing.T, raw uuid.UUID) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("mustUserID(%s): %v", raw, err)
	}
	return id
}

func mustRefreshTokenID(t *testing.T, raw uuid.UUID) domain.RefreshTokenID {
	t.Helper()
	id, err := domain.NewRefreshTokenID(raw)
	if err != nil {
		t.Fatalf("mustRefreshTokenID(%s): %v", raw, err)
	}
	return id
}
