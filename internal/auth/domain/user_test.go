package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func TestNewUser_Success(t *testing.T) {
	t.Parallel()

	id := mustUserID(t, uuid.New())
	email := mustEmail(t, "user@example.com")
	username := mustUsername(t, "user_1")
	hash := mustPasswordHash(t, "$2a$10$hash")
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	u, err := domain.NewUser(id, email, username, hash, createdAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID() != id {
		t.Fatalf("ID mismatch")
	}
	if u.Email() != email {
		t.Fatalf("Email mismatch")
	}
	if u.Username() != username {
		t.Fatalf("Username mismatch")
	}
	if u.PasswordHash() != hash {
		t.Fatalf("PasswordHash mismatch")
	}
	if !u.CreatedAt().Equal(createdAt) {
		t.Fatalf("CreatedAt mismatch")
	}
}

func TestNewUser_InvalidUserID(t *testing.T) {
	t.Parallel()

	_, err := domain.NewUser(
		domain.UserID{},
		mustEmail(t, "user@example.com"),
		mustUsername(t, "user_1"),
		mustPasswordHash(t, "$2a$10$hash"),
		time.Now(),
	)
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestNewUser_InvalidCreatedAt(t *testing.T) {
	t.Parallel()

	_, err := domain.NewUser(
		mustUserID(t, uuid.New()),
		mustEmail(t, "user@example.com"),
		mustUsername(t, "user_1"),
		mustPasswordHash(t, "$2a$10$hash"),
		time.Time{},
	)
	if !errors.Is(err, domain.ErrInvalidCreatedAt) {
		t.Fatalf("got %v, want ErrInvalidCreatedAt", err)
	}
}

func TestReconstructUser_Success(t *testing.T) {
	t.Parallel()

	id := mustUserID(t, uuid.New())
	email := mustEmail(t, "user@example.com")
	username := mustUsername(t, "user_1")
	hash := mustPasswordHash(t, "$2a$10$hash")
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	u, err := domain.ReconstructUser(id, email, username, hash, createdAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID() != id || !u.CreatedAt().Equal(createdAt) {
		t.Fatalf("ReconstructUser identity mismatch")
	}
}
