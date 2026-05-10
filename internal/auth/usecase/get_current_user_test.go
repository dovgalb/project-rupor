package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

func TestGetCurrentUser_Success(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	user := mustUser(t, "user@example.com", "user_1", "$2a$10$hash", now)
	users := newFakeUserRepo()
	users.byID[user.ID().String()] = user

	uc := usecase.NewGetCurrentUser(users)
	out, err := uc.Execute(context.Background(), usecase.GetCurrentUserInput{UserID: user.ID().String()})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.UserID != user.ID().String() || out.Email != user.Email().String() ||
		out.Username != user.Username().String() || !out.CreatedAt.Equal(now) {
		t.Fatalf("output mismatch: %+v", out)
	}
}

func TestGetCurrentUser_BadUUID(t *testing.T) {
	t.Parallel()

	uc := usecase.NewGetCurrentUser(newFakeUserRepo())
	_, err := uc.Execute(context.Background(), usecase.GetCurrentUserInput{UserID: "not-uuid"})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestGetCurrentUser_NotFound(t *testing.T) {
	t.Parallel()

	uc := usecase.NewGetCurrentUser(newFakeUserRepo())
	_, err := uc.Execute(context.Background(), usecase.GetCurrentUserInput{UserID: uuid.New().String()})
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("got %v, want ErrUserNotFound", err)
	}
}
