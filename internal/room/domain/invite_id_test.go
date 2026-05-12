package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestNewInviteID_Valid_Constructs(t *testing.T) {
	t.Parallel()

	raw := uuid.New()
	id, err := domain.NewInviteID(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.UUID() != raw {
		t.Fatalf("UUID() = %v, want %v", id.UUID(), raw)
	}
}

func TestNewInviteID_Zero_ReturnsErrInvalidInviteID(t *testing.T) {
	t.Parallel()

	_, err := domain.NewInviteID(uuid.Nil)
	if !errors.Is(err, domain.ErrInvalidInviteID) {
		t.Fatalf("got %v, want ErrInvalidInviteID", err)
	}
}
