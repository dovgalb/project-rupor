package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

type createRoomSUT struct {
	uc     *usecase.CreateRoom
	rooms  *fakeRoomRepo
	clock  *fixedClock
	uuids  *fixedUUID
	now    time.Time
	roomID uuid.UUID
	owner  uuid.UUID
}

func newCreateRoomSUT(t *testing.T) *createRoomSUT {
	t.Helper()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	rid := uuid.New()
	owner := uuid.New()
	rooms := newFakeRoomRepo()
	clock := &fixedClock{now: now}
	uuids := &fixedUUID{next: []uuid.UUID{rid}}
	return &createRoomSUT{
		uc:     usecase.NewCreateRoom(rooms, clock, uuids),
		rooms:  rooms,
		clock:  clock,
		uuids:  uuids,
		now:    now,
		roomID: rid,
		owner:  owner,
	}
}

func TestCreateRoom_Valid_PersistsRoomAndOwnerMembership(t *testing.T) {
	t.Parallel()

	sut := newCreateRoomSUT(t)
	out, err := sut.uc.Execute(context.Background(), usecase.CreateRoomInput{
		ActorID: sut.owner,
		Name:    "general",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Room.ID().UUID() != sut.roomID {
		t.Fatalf("Room.ID = %v, want %v", out.Room.ID().UUID(), sut.roomID)
	}
	if out.Room.OwnerID().UUID() != sut.owner {
		t.Fatalf("Room.OwnerID = %v, want %v", out.Room.OwnerID().UUID(), sut.owner)
	}
	if len(sut.rooms.saved) != 1 || len(sut.rooms.savedMemberships) != 1 {
		t.Fatalf("expected 1 room + 1 membership saved, got %d/%d",
			len(sut.rooms.saved), len(sut.rooms.savedMemberships))
	}
	if sut.rooms.savedMemberships[0].Role() != domain.RoleOwner {
		t.Fatalf("saved membership role = %v, want RoleOwner", sut.rooms.savedMemberships[0].Role())
	}
}

func TestCreateRoom_InvalidName_ReturnsErrInvalidRoomName(t *testing.T) {
	t.Parallel()

	sut := newCreateRoomSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.CreateRoomInput{
		ActorID: sut.owner,
		Name:    "",
	})
	if !errors.Is(err, domain.ErrInvalidRoomName) {
		t.Fatalf("got %v, want ErrInvalidRoomName", err)
	}
	if len(sut.rooms.saved) != 0 {
		t.Fatalf("repo touched on invalid name")
	}
}

func TestCreateRoom_RepoFails_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	sut := newCreateRoomSUT(t)
	repoErr := errors.New("db down")
	sut.rooms.saveWithOwnerErr = repoErr

	_, err := sut.uc.Execute(context.Background(), usecase.CreateRoomInput{
		ActorID: sut.owner,
		Name:    "general",
	})
	if !errors.Is(err, repoErr) {
		t.Fatalf("got %v, want wrapped repoErr", err)
	}
}

func TestCreateRoom_UsesInjectedClockAndUUID(t *testing.T) {
	t.Parallel()

	sut := newCreateRoomSUT(t)
	out, err := sut.uc.Execute(context.Background(), usecase.CreateRoomInput{
		ActorID: sut.owner,
		Name:    "general",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !out.Room.CreatedAt().Equal(sut.now) {
		t.Fatalf("CreatedAt = %v, want %v", out.Room.CreatedAt(), sut.now)
	}
	if out.Room.ID().UUID() != sut.roomID {
		t.Fatalf("ID = %v, want injected %v", out.Room.ID().UUID(), sut.roomID)
	}
}
