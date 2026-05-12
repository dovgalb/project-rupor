package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
	"github.com/dovgalb/project-rupor/internal/channel/usecase"
)

type createChannelSUT struct {
	uc         *usecase.CreateChannel
	channels   *fakeChannelRepo
	membership *fakeMembershipQuery
	clock      *fixedClock
	uuids      *fixedUUID
	now        time.Time
	actor      uuid.UUID
	room       uuid.UUID
	chanUUID   uuid.UUID
}

func newCreateChannelSUT(t *testing.T) *createChannelSUT {
	t.Helper()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	actor := uuid.New()
	room := uuid.New()
	chanUUID := uuid.New()
	channels := newFakeChannelRepo()
	membership := newFakeMembershipQuery()
	clock := &fixedClock{now: now}
	uuids := &fixedUUID{next: []uuid.UUID{chanUUID}}
	return &createChannelSUT{
		uc:         usecase.NewCreateChannel(channels, membership, clock, uuids),
		channels:   channels,
		membership: membership,
		clock:      clock,
		uuids:      uuids,
		now:        now,
		actor:      actor,
		room:       room,
		chanUUID:   chanUUID,
	}
}

func TestCreateChannel_AsAdmin_PersistsChannel(t *testing.T) {
	t.Parallel()

	sut := newCreateChannelSUT(t)
	sut.membership.withRole(sut.room, sut.actor, "admin")

	out, err := sut.uc.Execute(context.Background(), usecase.CreateChannelInput{
		ActorID: sut.actor,
		RoomID:  sut.room,
		Name:    "general",
		Kind:    "text",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Channel.ID().UUID() != sut.chanUUID {
		t.Fatalf("Channel.ID mismatch")
	}
	if len(sut.channels.saved) != 1 {
		t.Fatalf("saved = %d, want 1", len(sut.channels.saved))
	}
}

func TestCreateChannel_AsOwner_PersistsChannel(t *testing.T) {
	t.Parallel()

	sut := newCreateChannelSUT(t)
	sut.membership.withRole(sut.room, sut.actor, "owner")

	_, err := sut.uc.Execute(context.Background(), usecase.CreateChannelInput{
		ActorID: sut.actor,
		RoomID:  sut.room,
		Name:    "general",
		Kind:    "voice",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestCreateChannel_InvalidName_ReturnsErrInvalidChannelName(t *testing.T) {
	t.Parallel()

	sut := newCreateChannelSUT(t)
	sut.membership.withRole(sut.room, sut.actor, "admin")

	_, err := sut.uc.Execute(context.Background(), usecase.CreateChannelInput{
		ActorID: sut.actor,
		RoomID:  sut.room,
		Name:    "",
		Kind:    "text",
	})
	if !errors.Is(err, domain.ErrInvalidChannelName) {
		t.Fatalf("got %v, want ErrInvalidChannelName", err)
	}
}

func TestCreateChannel_InvalidKind_ReturnsErrInvalidChannelKind(t *testing.T) {
	t.Parallel()

	sut := newCreateChannelSUT(t)
	sut.membership.withRole(sut.room, sut.actor, "admin")

	_, err := sut.uc.Execute(context.Background(), usecase.CreateChannelInput{
		ActorID: sut.actor,
		RoomID:  sut.room,
		Name:    "general",
		Kind:    "video",
	})
	if !errors.Is(err, domain.ErrInvalidChannelKind) {
		t.Fatalf("got %v, want ErrInvalidChannelKind", err)
	}
}

func TestCreateChannel_NotMember_ReturnsErrAccessDenied(t *testing.T) {
	t.Parallel()

	sut := newCreateChannelSUT(t)
	// никаких withRole — fakeMembershipQuery вернёт ErrChannelAccessDenied
	_, err := sut.uc.Execute(context.Background(), usecase.CreateChannelInput{
		ActorID: sut.actor,
		RoomID:  sut.room,
		Name:    "general",
		Kind:    "text",
	})
	if !errors.Is(err, domain.ErrChannelAccessDenied) {
		t.Fatalf("got %v, want ErrChannelAccessDenied", err)
	}
	if len(sut.channels.saved) != 0 {
		t.Fatalf("repo touched on access denied")
	}
}

func TestCreateChannel_AsMember_ReturnsErrInsufficientRole(t *testing.T) {
	t.Parallel()

	sut := newCreateChannelSUT(t)
	sut.membership.withRole(sut.room, sut.actor, "member")

	_, err := sut.uc.Execute(context.Background(), usecase.CreateChannelInput{
		ActorID: sut.actor,
		RoomID:  sut.room,
		Name:    "general",
		Kind:    "text",
	})
	if !errors.Is(err, domain.ErrChannelInsufficientRole) {
		t.Fatalf("got %v, want ErrChannelInsufficientRole", err)
	}
}

func TestCreateChannel_DuplicateName_ReturnsErrChannelNameAlreadyTaken(t *testing.T) {
	t.Parallel()

	sut := newCreateChannelSUT(t)
	sut.membership.withRole(sut.room, sut.actor, "admin")
	// предзаполнить канал с этим именем
	sut.channels.put(mustChannel(t, uuid.New(), sut.room, "general", domain.ChannelKindText, sut.now))

	_, err := sut.uc.Execute(context.Background(), usecase.CreateChannelInput{
		ActorID: sut.actor,
		RoomID:  sut.room,
		Name:    "general",
		Kind:    "text",
	})
	if !errors.Is(err, domain.ErrChannelNameAlreadyTaken) {
		t.Fatalf("got %v, want ErrChannelNameAlreadyTaken", err)
	}
}
