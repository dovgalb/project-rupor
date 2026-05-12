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

type getRoomSUT struct {
	uc          *usecase.GetRoom
	rooms       *fakeRoomRepo
	memberships *fakeMembershipRepo
	now         time.Time
}

func newGetRoomSUT(t *testing.T) *getRoomSUT {
	t.Helper()
	rooms := newFakeRoomRepo()
	memberships := newFakeMembershipRepo()
	return &getRoomSUT{
		uc:          usecase.NewGetRoom(rooms, memberships),
		rooms:       rooms,
		memberships: memberships,
		now:         time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
	}
}

func TestGetRoom_AsMember_ReturnsRoom(t *testing.T) {
	t.Parallel()

	sut := newGetRoomSUT(t)
	ownerID := uuid.New()
	roomUUID := uuid.New()
	room := mustRoom(t, roomUUID, ownerID, "general", sut.now)
	sut.rooms.rooms[roomUUID] = room
	sut.memberships.put(mustMembership(t, roomUUID, ownerID, domain.RoleMember, sut.now))

	out, err := sut.uc.Execute(context.Background(), usecase.GetRoomInput{
		ActorID: ownerID,
		RoomID:  roomUUID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Room.ID().UUID() != roomUUID {
		t.Fatalf("ID mismatch")
	}
}

func TestGetRoom_NotMember_ReturnsErrNotMember(t *testing.T) {
	t.Parallel()

	sut := newGetRoomSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.GetRoomInput{
		ActorID: uuid.New(),
		RoomID:  uuid.New(),
	})
	if !errors.Is(err, domain.ErrNotMember) {
		t.Fatalf("got %v, want ErrNotMember", err)
	}
}

func TestGetRoom_RoomDeletedAfterMembershipCheck_ReturnsErrRoomNotFound(t *testing.T) {
	t.Parallel()

	sut := newGetRoomSUT(t)
	actor := uuid.New()
	roomUUID := uuid.New()
	// Membership есть, но room в репозитории нет — эмулирует race delete.
	sut.memberships.put(mustMembership(t, roomUUID, actor, domain.RoleMember, sut.now))

	_, err := sut.uc.Execute(context.Background(), usecase.GetRoomInput{
		ActorID: actor,
		RoomID:  roomUUID,
	})
	if !errors.Is(err, domain.ErrRoomNotFound) {
		t.Fatalf("got %v, want ErrRoomNotFound", err)
	}
}
