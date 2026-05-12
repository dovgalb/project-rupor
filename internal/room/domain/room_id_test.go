package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestNewRoomID_Valid_Constructs(t *testing.T) {
	t.Parallel()

	raw := uuid.New()
	id, err := domain.NewRoomID(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.UUID() != raw {
		t.Fatalf("UUID() = %v, want %v", id.UUID(), raw)
	}
}

func TestNewRoomID_Zero_ReturnsErrInvalidRoomID(t *testing.T) {
	t.Parallel()

	_, err := domain.NewRoomID(uuid.Nil)
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestRoomID_IsZero_ZeroValue(t *testing.T) {
	t.Parallel()

	var id domain.RoomID
	if !id.IsZero() {
		t.Fatalf("zero RoomID should be IsZero()")
	}
}
