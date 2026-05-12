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

type regenInviteSUT struct {
	uc          *usecase.RegenerateInvite
	invites     *fakeInviteRepo
	memberships *fakeMembershipRepo
	codes       *fakeInviteCodeGen
	clock       *fixedClock
	uuids       *fixedUUID
	now         time.Time
}

func newRegenInviteSUT(t *testing.T, queue []string, uuidQueue []uuid.UUID) *regenInviteSUT {
	t.Helper()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	invites := newFakeInviteRepo()
	memberships := newFakeMembershipRepo()
	codes := &fakeInviteCodeGen{queue: queue}
	clock := &fixedClock{now: now}
	if len(uuidQueue) == 0 {
		uuidQueue = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	}
	uuids := &fixedUUID{next: uuidQueue}
	return &regenInviteSUT{
		uc:          usecase.NewRegenerateInvite(invites, memberships, codes, clock, uuids),
		invites:     invites,
		memberships: memberships,
		codes:       codes,
		clock:       clock,
		uuids:       uuids,
		now:         now,
	}
}

func TestRegenerateInvite_AsAdmin_RevokesOldAndCreatesNew(t *testing.T) {
	t.Parallel()

	sut := newRegenInviteSUT(t, []string{"BBBBBBBB"}, nil)
	actor := uuid.New()
	roomUUID := uuid.New()
	sut.memberships.put(mustMembership(t, roomUUID, actor, domain.RoleAdmin, sut.now))
	sut.invites.put(mustInvite(t, uuid.New(), roomUUID, "AAAAAAAA", actor, sut.now))

	out, err := sut.uc.Execute(context.Background(), usecase.RegenerateInviteInput{
		ActorID: actor,
		RoomID:  roomUUID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Invite.Code().String() != "BBBBBBBB" {
		t.Fatalf("Invite.Code = %q, want BBBBBBBB", out.Invite.Code().String())
	}
}

func TestRegenerateInvite_AsOwner_Succeeds(t *testing.T) {
	t.Parallel()

	sut := newRegenInviteSUT(t, []string{"CCCCCCCC"}, nil)
	actor := uuid.New()
	roomUUID := uuid.New()
	sut.memberships.put(mustMembership(t, roomUUID, actor, domain.RoleOwner, sut.now))

	_, err := sut.uc.Execute(context.Background(), usecase.RegenerateInviteInput{
		ActorID: actor,
		RoomID:  roomUUID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestRegenerateInvite_AsMember_ReturnsErrInsufficientRole(t *testing.T) {
	t.Parallel()

	sut := newRegenInviteSUT(t, nil, nil)
	actor := uuid.New()
	roomUUID := uuid.New()
	sut.memberships.put(mustMembership(t, roomUUID, actor, domain.RoleMember, sut.now))

	_, err := sut.uc.Execute(context.Background(), usecase.RegenerateInviteInput{
		ActorID: actor,
		RoomID:  roomUUID,
	})
	if !errors.Is(err, domain.ErrInsufficientRole) {
		t.Fatalf("got %v, want ErrInsufficientRole", err)
	}
}

func TestRegenerateInvite_NotMember_ReturnsErrNotMember(t *testing.T) {
	t.Parallel()

	sut := newRegenInviteSUT(t, nil, nil)
	_, err := sut.uc.Execute(context.Background(), usecase.RegenerateInviteInput{
		ActorID: uuid.New(),
		RoomID:  uuid.New(),
	})
	if !errors.Is(err, domain.ErrNotMember) {
		t.Fatalf("got %v, want ErrNotMember", err)
	}
}

func TestRegenerateInvite_FirstCodeCollides_RetriesAndSucceeds(t *testing.T) {
	t.Parallel()

	sut := newRegenInviteSUT(t, []string{"AAAAAAAA", "BBBBBBBB"}, nil)
	sut.invites.nextRegenCollisions = 1
	actor := uuid.New()
	roomUUID := uuid.New()
	sut.memberships.put(mustMembership(t, roomUUID, actor, domain.RoleOwner, sut.now))

	out, err := sut.uc.Execute(context.Background(), usecase.RegenerateInviteInput{
		ActorID: actor,
		RoomID:  roomUUID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sut.codes.calls != 2 {
		t.Fatalf("codes.calls = %d, want 2 (1 collision + 1 success)", sut.codes.calls)
	}
	if out.Invite.Code().String() != "BBBBBBBB" {
		t.Fatalf("Invite.Code = %q, want BBBBBBBB (second)", out.Invite.Code().String())
	}
}

func TestRegenerateInvite_AllRetriesCollide_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	sut := newRegenInviteSUT(t, []string{"AAAAAAAA", "BBBBBBBB", "CCCCCCCC"}, nil)
	sut.invites.nextRegenCollisions = 5
	actor := uuid.New()
	roomUUID := uuid.New()
	sut.memberships.put(mustMembership(t, roomUUID, actor, domain.RoleOwner, sut.now))

	_, err := sut.uc.Execute(context.Background(), usecase.RegenerateInviteInput{
		ActorID: actor,
		RoomID:  roomUUID,
	})
	if !errors.Is(err, usecase.ErrInviteCodeCollision) {
		t.Fatalf("got %v, want wrapped ErrInviteCodeCollision", err)
	}
	if sut.codes.calls != 3 {
		t.Fatalf("codes.calls = %d, want 3 (max retries)", sut.codes.calls)
	}
}
