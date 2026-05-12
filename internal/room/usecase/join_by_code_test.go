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

type joinByCodeSUT struct {
	uc          *usecase.JoinByCode
	invites     *fakeInviteRepo
	memberships *fakeMembershipRepo
	rooms       *fakeRoomRepo
	clock       *fixedClock
	events      *fakeRoomEventsPublisher
	now         time.Time
}

func newJoinByCodeSUT(t *testing.T) *joinByCodeSUT {
	t.Helper()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	invites := newFakeInviteRepo()
	memberships := newFakeMembershipRepo()
	rooms := newFakeRoomRepo()
	clock := &fixedClock{now: now}
	events := newFakeRoomEventsPublisher()
	return &joinByCodeSUT{
		uc:          usecase.NewJoinByCode(invites, memberships, rooms, clock, events),
		invites:     invites,
		memberships: memberships,
		rooms:       rooms,
		clock:       clock,
		events:      events,
		now:         now,
	}
}

func TestJoinByCode_NewMember_AddsMembershipAndReturnsRoom(t *testing.T) {
	t.Parallel()

	sut := newJoinByCodeSUT(t)
	actor := uuid.New()
	owner := uuid.New()
	roomUUID := uuid.New()
	sut.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, owner, "r", sut.now)
	sut.invites.put(mustInvite(t, uuid.New(), roomUUID, "ABCDEFGH", owner, sut.now))

	out, err := sut.uc.Execute(context.Background(), usecase.JoinByCodeInput{
		ActorID: actor,
		Code:    "ABCDEFGH",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Room.ID().UUID() != roomUUID {
		t.Fatalf("Room.ID mismatch")
	}
	if len(sut.memberships.added) != 1 {
		t.Fatalf("memberships.added = %d, want 1", len(sut.memberships.added))
	}
	if sut.memberships.added[0].Role() != domain.RoleMember {
		t.Fatalf("added role = %v, want RoleMember", sut.memberships.added[0].Role())
	}
}

func TestJoinByCode_AlreadyMember_ReturnsErrAlreadyMember(t *testing.T) {
	t.Parallel()

	sut := newJoinByCodeSUT(t)
	actor := uuid.New()
	owner := uuid.New()
	roomUUID := uuid.New()
	sut.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, owner, "r", sut.now)
	sut.invites.put(mustInvite(t, uuid.New(), roomUUID, "ABCDEFGH", owner, sut.now))
	sut.memberships.put(mustMembership(t, roomUUID, actor, domain.RoleMember, sut.now))

	_, err := sut.uc.Execute(context.Background(), usecase.JoinByCodeInput{
		ActorID: actor,
		Code:    "ABCDEFGH",
	})
	if !errors.Is(err, domain.ErrAlreadyMember) {
		t.Fatalf("got %v, want ErrAlreadyMember", err)
	}
}

func TestJoinByCode_NoActiveInvite_ReturnsErrInviteNotFound(t *testing.T) {
	t.Parallel()

	sut := newJoinByCodeSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.JoinByCodeInput{
		ActorID: uuid.New(),
		Code:    "ABCDEFGH",
	})
	if !errors.Is(err, domain.ErrInviteNotFound) {
		t.Fatalf("got %v, want ErrInviteNotFound", err)
	}
}

func TestJoinByCode_InvalidFormat_ReturnsErrInvalidInviteCode(t *testing.T) {
	t.Parallel()

	sut := newJoinByCodeSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.JoinByCodeInput{
		ActorID: uuid.New(),
		Code:    "ABC",
	})
	if !errors.Is(err, domain.ErrInvalidInviteCode) {
		t.Fatalf("got %v, want ErrInvalidInviteCode", err)
	}
}

func TestJoinByCode_RaceWithDuplicateInsert_ReturnsErrAlreadyMember(t *testing.T) {
	t.Parallel()

	sut := newJoinByCodeSUT(t)
	actor := uuid.New()
	owner := uuid.New()
	roomUUID := uuid.New()
	sut.invites.put(mustInvite(t, uuid.New(), roomUUID, "ABCDEFGH", owner, sut.now))
	// FindByPair вернёт ErrNotMember, но Add вернёт ErrAlreadyMember (race).
	sut.memberships.addErr = domain.ErrAlreadyMember

	_, err := sut.uc.Execute(context.Background(), usecase.JoinByCodeInput{
		ActorID: actor,
		Code:    "ABCDEFGH",
	})
	if !errors.Is(err, domain.ErrAlreadyMember) {
		t.Fatalf("got %v, want ErrAlreadyMember", err)
	}
}

func TestJoinByCode_Success_PublishesMemberJoined(t *testing.T) {
	t.Parallel()

	sut := newJoinByCodeSUT(t)
	actor := uuid.New()
	owner := uuid.New()
	roomUUID := uuid.New()
	sut.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, owner, "r", sut.now)
	sut.invites.put(mustInvite(t, uuid.New(), roomUUID, "ABCDEFGH", owner, sut.now))

	_, err := sut.uc.Execute(context.Background(), usecase.JoinByCodeInput{
		ActorID: actor,
		Code:    "ABCDEFGH",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	calls := sut.events.calls()
	if len(calls) != 1 {
		t.Fatalf("publisher calls = %d, want 1", len(calls))
	}
	if calls[0].RoomID.UUID() != roomUUID {
		t.Fatalf("publish RoomID = %v, want %v", calls[0].RoomID, roomUUID)
	}
	if calls[0].UserID.UUID() != actor {
		t.Fatalf("publish UserID = %v, want %v", calls[0].UserID, actor)
	}
	if !calls[0].JoinedAt.Equal(sut.now) {
		t.Fatalf("publish JoinedAt = %v, want %v", calls[0].JoinedAt, sut.now)
	}
}

func TestJoinByCode_PublishPanic_DoesNotRollbackMembership(t *testing.T) {
	t.Parallel()

	sut := newJoinByCodeSUT(t)
	sut.events.panicNext = true
	actor := uuid.New()
	owner := uuid.New()
	roomUUID := uuid.New()
	sut.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, owner, "r", sut.now)
	sut.invites.put(mustInvite(t, uuid.New(), roomUUID, "ABCDEFGH", owner, sut.now))

	_, err := sut.uc.Execute(context.Background(), usecase.JoinByCodeInput{
		ActorID: actor,
		Code:    "ABCDEFGH",
	})
	if err != nil {
		t.Fatalf("Execute: %v (panic в publisher должна быть проглочена)", err)
	}
	if len(sut.memberships.added) != 1 {
		t.Fatalf("memberships.added = %d, want 1 (rollback не должен произойти)", len(sut.memberships.added))
	}
}

func TestJoinByCode_InvalidCode_DoesNotPublish(t *testing.T) {
	t.Parallel()

	sut := newJoinByCodeSUT(t)

	_, err := sut.uc.Execute(context.Background(), usecase.JoinByCodeInput{
		ActorID: uuid.New(),
		Code:    "ABCDEFGH", // валидный формат, но в fake-репо такого invite'а нет
	})
	if !errors.Is(err, domain.ErrInviteNotFound) {
		t.Fatalf("got %v, want ErrInviteNotFound", err)
	}
	if len(sut.events.calls()) != 0 {
		t.Fatalf("publisher вызван при ошибке Find")
	}
}
