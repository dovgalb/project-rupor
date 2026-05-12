package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestNewRoom_Valid_Constructs(t *testing.T) {
	t.Parallel()

	id := mustRoomID(t, uuid.New())
	ownerID := mustUserID(t, uuid.New())
	name := mustRoomName(t, "general")
	createdAt := defaultBuilderTime

	r, err := domain.NewRoom(id, ownerID, name, createdAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ID() != id {
		t.Fatalf("ID mismatch")
	}
	if r.OwnerID() != ownerID {
		t.Fatalf("OwnerID mismatch")
	}
	if r.Name() != name {
		t.Fatalf("Name mismatch")
	}
	if !r.CreatedAt().Equal(createdAt) {
		t.Fatalf("CreatedAt mismatch")
	}
}

func TestNewRoom_ZeroID_ReturnsErrInvalidRoomID(t *testing.T) {
	t.Parallel()

	_, err := domain.NewRoom(
		domain.RoomID{},
		mustUserID(t, uuid.New()),
		mustRoomName(t, "general"),
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestNewRoom_ZeroOwnerID_ReturnsErrInvalidUserID(t *testing.T) {
	t.Parallel()

	_, err := domain.NewRoom(
		mustRoomID(t, uuid.New()),
		domain.UserID{},
		mustRoomName(t, "general"),
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestNewRoom_ZeroCreatedAt_ReturnsErrInvalidCreatedAt(t *testing.T) {
	t.Parallel()

	_, err := domain.NewRoom(
		mustRoomID(t, uuid.New()),
		mustUserID(t, uuid.New()),
		mustRoomName(t, "general"),
		time.Time{},
	)
	if !errors.Is(err, domain.ErrInvalidCreatedAt) {
		t.Fatalf("got %v, want ErrInvalidCreatedAt", err)
	}
}

func TestReconstructRoom_Valid_Constructs(t *testing.T) {
	t.Parallel()

	id := mustRoomID(t, uuid.New())
	ownerID := mustUserID(t, uuid.New())
	name := mustRoomName(t, "general")

	r, err := domain.ReconstructRoom(id, ownerID, name, defaultBuilderTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ID() != id {
		t.Fatalf("ID mismatch")
	}
}

func TestRoom_TransferOwnership_ChangesOwner(t *testing.T) {
	t.Parallel()

	r := NewRoomBuilder(t).Build()
	newOwner := mustUserID(t, uuid.New())

	if err := r.TransferOwnership(newOwner); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.OwnerID() != newOwner {
		t.Fatalf("OwnerID = %v, want %v", r.OwnerID(), newOwner)
	}
}

func TestRoom_TransferOwnership_ZeroNewOwner_ReturnsErr(t *testing.T) {
	t.Parallel()

	r := NewRoomBuilder(t).Build()
	originalOwner := r.OwnerID()

	err := r.TransferOwnership(domain.UserID{})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
	if r.OwnerID() != originalOwner {
		t.Fatalf("owner mutated after failed transfer: got %v, want %v", r.OwnerID(), originalOwner)
	}
}
