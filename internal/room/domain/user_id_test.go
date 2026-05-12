package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestNewUserID_Valid_Constructs(t *testing.T) {
	t.Parallel()

	raw := uuid.New()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.UUID() != raw {
		t.Fatalf("UUID() = %v, want %v", id.UUID(), raw)
	}
}

func TestNewUserID_Zero_ReturnsErrInvalidUserID(t *testing.T) {
	t.Parallel()

	_, err := domain.NewUserID(uuid.Nil)
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}
