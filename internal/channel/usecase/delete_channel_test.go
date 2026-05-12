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

type deleteChannelSUT struct {
	uc         *usecase.DeleteChannel
	channels   *fakeChannelRepo
	membership *fakeMembershipQuery
	now        time.Time
}

func newDeleteChannelSUT(t *testing.T) *deleteChannelSUT {
	t.Helper()
	channels := newFakeChannelRepo()
	membership := newFakeMembershipQuery()
	return &deleteChannelSUT{
		uc:         usecase.NewDeleteChannel(channels, membership),
		channels:   channels,
		membership: membership,
		now:        time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
	}
}

func TestDeleteChannel_AsAdmin_DeletesChannel(t *testing.T) {
	t.Parallel()

	sut := newDeleteChannelSUT(t)
	actor := uuid.New()
	room := uuid.New()
	chUUID := uuid.New()
	sut.membership.withRole(room, actor, "admin")
	sut.channels.put(mustChannel(t, chUUID, room, "general", domain.ChannelKindText, sut.now))

	err := sut.uc.Execute(context.Background(), usecase.DeleteChannelInput{
		ActorID:   actor,
		RoomID:    room,
		ChannelID: chUUID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(sut.channels.deleted) != 1 {
		t.Fatalf("deleted = %d, want 1", len(sut.channels.deleted))
	}
}

func TestDeleteChannel_AsMember_ReturnsErrInsufficientRole(t *testing.T) {
	t.Parallel()

	sut := newDeleteChannelSUT(t)
	actor := uuid.New()
	room := uuid.New()
	sut.membership.withRole(room, actor, "member")

	err := sut.uc.Execute(context.Background(), usecase.DeleteChannelInput{
		ActorID:   actor,
		RoomID:    room,
		ChannelID: uuid.New(),
	})
	if !errors.Is(err, domain.ErrChannelInsufficientRole) {
		t.Fatalf("got %v, want ErrChannelInsufficientRole", err)
	}
}

func TestDeleteChannel_NotMember_ReturnsErrAccessDenied(t *testing.T) {
	t.Parallel()

	sut := newDeleteChannelSUT(t)
	err := sut.uc.Execute(context.Background(), usecase.DeleteChannelInput{
		ActorID:   uuid.New(),
		RoomID:    uuid.New(),
		ChannelID: uuid.New(),
	})
	if !errors.Is(err, domain.ErrChannelAccessDenied) {
		t.Fatalf("got %v, want ErrChannelAccessDenied", err)
	}
}

func TestDeleteChannel_NotFound_ReturnsErrChannelNotFound(t *testing.T) {
	t.Parallel()

	sut := newDeleteChannelSUT(t)
	actor := uuid.New()
	room := uuid.New()
	sut.membership.withRole(room, actor, "admin")

	err := sut.uc.Execute(context.Background(), usecase.DeleteChannelInput{
		ActorID:   actor,
		RoomID:    room,
		ChannelID: uuid.New(),
	})
	if !errors.Is(err, domain.ErrChannelNotFound) {
		t.Fatalf("got %v, want ErrChannelNotFound", err)
	}
}

// TestDeleteChannel_WrongRoomID — channel существует, но в другой комнате.
// fakeChannelRepo проверяет принадлежность (channelID, roomID) — это инвариант сигнатуры порта,
// защищающий от подмены roomID в URL.
func TestDeleteChannel_WrongRoomID_ReturnsErrChannelNotFound(t *testing.T) {
	t.Parallel()

	sut := newDeleteChannelSUT(t)
	actor := uuid.New()
	realRoom := uuid.New()
	otherRoom := uuid.New()
	chUUID := uuid.New()
	sut.membership.withRole(otherRoom, actor, "admin")
	sut.channels.put(mustChannel(t, chUUID, realRoom, "g", domain.ChannelKindText, sut.now))

	err := sut.uc.Execute(context.Background(), usecase.DeleteChannelInput{
		ActorID:   actor,
		RoomID:    otherRoom,
		ChannelID: chUUID,
	})
	if !errors.Is(err, domain.ErrChannelNotFound) {
		t.Fatalf("got %v, want ErrChannelNotFound", err)
	}
}
