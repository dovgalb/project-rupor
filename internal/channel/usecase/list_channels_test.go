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

func TestListChannels_AsMember_ReturnsAll(t *testing.T) {
	t.Parallel()

	channels := newFakeChannelRepo()
	membership := newFakeMembershipQuery()
	uc := usecase.NewListChannels(channels, membership)
	actor := uuid.New()
	room := uuid.New()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	membership.withRole(room, actor, "member")

	channels.put(mustChannel(t, uuid.New(), room, "general", domain.ChannelKindText, now))
	channels.put(mustChannel(t, uuid.New(), room, "voice", domain.ChannelKindVoice, now))
	channels.put(mustChannel(t, uuid.New(), uuid.New(), "other-room-chan", domain.ChannelKindText, now))

	out, err := uc.Execute(context.Background(), usecase.ListChannelsInput{
		ActorID: actor,
		RoomID:  room,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("Items = %d, want 2 (other-room filtered)", len(out.Items))
	}
}

func TestListChannels_AsOwner_ReturnsAll(t *testing.T) {
	t.Parallel()

	channels := newFakeChannelRepo()
	membership := newFakeMembershipQuery()
	uc := usecase.NewListChannels(channels, membership)
	actor := uuid.New()
	room := uuid.New()
	membership.withRole(room, actor, "owner")
	channels.put(mustChannel(t, uuid.New(), room, "g", domain.ChannelKindText, time.Now()))

	out, err := uc.Execute(context.Background(), usecase.ListChannelsInput{
		ActorID: actor,
		RoomID:  room,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != 1 {
		t.Fatalf("Items = %d, want 1", len(out.Items))
	}
}

func TestListChannels_NotMember_ReturnsErrAccessDenied(t *testing.T) {
	t.Parallel()

	channels := newFakeChannelRepo()
	membership := newFakeMembershipQuery()
	uc := usecase.NewListChannels(channels, membership)

	_, err := uc.Execute(context.Background(), usecase.ListChannelsInput{
		ActorID: uuid.New(),
		RoomID:  uuid.New(),
	})
	if !errors.Is(err, domain.ErrChannelAccessDenied) {
		t.Fatalf("got %v, want ErrChannelAccessDenied", err)
	}
}

func TestListChannels_EmptyRoom_ReturnsEmptyItems(t *testing.T) {
	t.Parallel()

	channels := newFakeChannelRepo()
	membership := newFakeMembershipQuery()
	uc := usecase.NewListChannels(channels, membership)
	actor := uuid.New()
	room := uuid.New()
	membership.withRole(room, actor, "member")

	out, err := uc.Execute(context.Background(), usecase.ListChannelsInput{
		ActorID: actor,
		RoomID:  room,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != 0 {
		t.Fatalf("Items = %d, want 0", len(out.Items))
	}
}
