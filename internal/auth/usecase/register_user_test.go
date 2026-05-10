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

type registerSUT struct {
	uc      *usecase.RegisterUser
	users   *fakeUserRepo
	hasher  *fakeHasher
	clock   *fixedClock
	uuids   *fixedUUID
	now     time.Time
	userUID uuid.UUID
}

func newRegisterSUT(t *testing.T) *registerSUT {
	t.Helper()
	uid := uuid.New()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	users := newFakeUserRepo()
	hasher := newFakeHasher()
	clock := &fixedClock{now: now}
	uuids := &fixedUUID{next: []uuid.UUID{uid}}
	return &registerSUT{
		uc:      usecase.NewRegisterUser(users, hasher, clock, uuids),
		users:   users,
		hasher:  hasher,
		clock:   clock,
		uuids:   uuids,
		now:     now,
		userUID: uid,
	}
}

func TestRegisterUser_Success(t *testing.T) {
	t.Parallel()

	sut := newRegisterSUT(t)
	out, err := sut.uc.Execute(context.Background(), usecase.RegisterUserInput{
		Email:    "User@Example.com",
		Username: "user_1",
		Password: "12345678",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.UserID != sut.userUID.String() {
		t.Fatalf("UserID = %q, want %q", out.UserID, sut.userUID.String())
	}
	if out.Email != "user@example.com" {
		t.Fatalf("Email = %q", out.Email)
	}
	if out.Username != "user_1" {
		t.Fatalf("Username = %q", out.Username)
	}
	if !out.CreatedAt.Equal(sut.now) {
		t.Fatalf("CreatedAt = %s", out.CreatedAt)
	}
	if len(sut.users.saved) != 1 {
		t.Fatalf("expected 1 saved user, got %d", len(sut.users.saved))
	}
}

func TestRegisterUser_InvalidEmail(t *testing.T) {
	t.Parallel()

	sut := newRegisterSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.RegisterUserInput{
		Email:    "not-email",
		Username: "user_1",
		Password: "12345678",
	})
	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Fatalf("got %v, want ErrInvalidEmail", err)
	}
	if len(sut.users.saved) != 0 || sut.hasher.verifyCalls != 0 {
		t.Fatalf("repo or hasher were touched")
	}
}

func TestRegisterUser_InvalidUsername(t *testing.T) {
	t.Parallel()

	sut := newRegisterSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.RegisterUserInput{
		Email:    "user@example.com",
		Username: "ab",
		Password: "12345678",
	})
	if !errors.Is(err, domain.ErrInvalidUsername) {
		t.Fatalf("got %v, want ErrInvalidUsername", err)
	}
}

func TestRegisterUser_InvalidPassword(t *testing.T) {
	t.Parallel()

	sut := newRegisterSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.RegisterUserInput{
		Email:    "user@example.com",
		Username: "user_1",
		Password: "1234",
	})
	if !errors.Is(err, domain.ErrInvalidPassword) {
		t.Fatalf("got %v, want ErrInvalidPassword", err)
	}
	if sut.hasher.verifyCalls != 0 {
		t.Fatalf("hasher.Verify was called")
	}
}

func TestRegisterUser_EmailTaken(t *testing.T) {
	t.Parallel()

	sut := newRegisterSUT(t)
	sut.users.saveErr = domain.ErrEmailAlreadyTaken

	_, err := sut.uc.Execute(context.Background(), usecase.RegisterUserInput{
		Email:    "user@example.com",
		Username: "user_1",
		Password: "12345678",
	})
	if !errors.Is(err, domain.ErrEmailAlreadyTaken) {
		t.Fatalf("got %v, want ErrEmailAlreadyTaken", err)
	}
}

func TestRegisterUser_UsernameTaken(t *testing.T) {
	t.Parallel()

	sut := newRegisterSUT(t)
	sut.users.saveErr = domain.ErrUsernameAlreadyTaken

	_, err := sut.uc.Execute(context.Background(), usecase.RegisterUserInput{
		Email:    "user@example.com",
		Username: "user_1",
		Password: "12345678",
	})
	if !errors.Is(err, domain.ErrUsernameAlreadyTaken) {
		t.Fatalf("got %v, want ErrUsernameAlreadyTaken", err)
	}
}
