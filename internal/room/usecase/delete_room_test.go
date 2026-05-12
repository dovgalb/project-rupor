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

type deleteRoomSUT struct {
	uc          *usecase.DeleteRoom
	rooms       *fakeRoomRepo
	memberships *fakeMembershipRepo
	now         time.Time
}

func newDeleteRoomSUT(t *testing.T) *deleteRoomSUT {
	t.Helper()
	rooms := newFakeRoomRepo()
	memberships := newFakeMembershipRepo()
	return &deleteRoomSUT{
		uc:          usecase.NewDeleteRoom(rooms, memberships),
		rooms:       rooms,
		memberships: memberships,
		now:         time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
	}
}

func TestDeleteRoom_AsOwner_DeletesRoom(t *testing.T) {
	t.Parallel()

	sut := newDeleteRoomSUT(t)
	owner := uuid.New()
	roomUUID := uuid.New()
	sut.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, owner, "r", sut.now)
	sut.memberships.put(mustMembership(t, roomUUID, owner, domain.RoleOwner, sut.now))

	err := sut.uc.Execute(context.Background(), usecase.DeleteRoomInput{
		ActorID: owner,
		RoomID:  roomUUID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(sut.rooms.deleted) != 1 {
		t.Fatalf("deletes = %d, want 1", len(sut.rooms.deleted))
	}
}

func TestDeleteRoom_AsAdmin_ReturnsErrInsufficientRole(t *testing.T) {
	t.Parallel()

	sut := newDeleteRoomSUT(t)
	actor := uuid.New()
	roomUUID := uuid.New()
	sut.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, uuid.New(), "r", sut.now)
	sut.memberships.put(mustMembership(t, roomUUID, actor, domain.RoleAdmin, sut.now))

	err := sut.uc.Execute(context.Background(), usecase.DeleteRoomInput{
		ActorID: actor,
		RoomID:  roomUUID,
	})
	if !errors.Is(err, domain.ErrInsufficientRole) {
		t.Fatalf("got %v, want ErrInsufficientRole", err)
	}
	if len(sut.rooms.deleted) != 0 {
		t.Fatalf("deleted on insufficient role")
	}
}

func TestDeleteRoom_AsMember_ReturnsErrInsufficientRole(t *testing.T) {
	t.Parallel()

	sut := newDeleteRoomSUT(t)
	actor := uuid.New()
	roomUUID := uuid.New()
	sut.memberships.put(mustMembership(t, roomUUID, actor, domain.RoleMember, sut.now))

	err := sut.uc.Execute(context.Background(), usecase.DeleteRoomInput{
		ActorID: actor,
		RoomID:  roomUUID,
	})
	if !errors.Is(err, domain.ErrInsufficientRole) {
		t.Fatalf("got %v, want ErrInsufficientRole", err)
	}
}

func TestDeleteRoom_NotMember_ReturnsErrNotMember(t *testing.T) {
	t.Parallel()

	sut := newDeleteRoomSUT(t)
	err := sut.uc.Execute(context.Background(), usecase.DeleteRoomInput{
		ActorID: uuid.New(),
		RoomID:  uuid.New(),
	})
	if !errors.Is(err, domain.ErrNotMember) {
		t.Fatalf("got %v, want ErrNotMember", err)
	}
}
