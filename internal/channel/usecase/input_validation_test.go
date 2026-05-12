package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
	"github.com/dovgalb/project-rupor/internal/channel/usecase"
)

// Покрытие парсинг-веток ID-VO на границе use case (до Membership.Require).

func TestCreateChannel_ZeroActorID_ReturnsErr(t *testing.T) {
	t.Parallel()
	sut := newCreateChannelSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.CreateChannelInput{
		ActorID: uuid.Nil,
		RoomID:  uuid.New(),
		Name:    "general",
		Kind:    "text",
	})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestCreateChannel_ZeroRoomID_ReturnsErr(t *testing.T) {
	t.Parallel()
	sut := newCreateChannelSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.CreateChannelInput{
		ActorID: uuid.New(),
		RoomID:  uuid.Nil,
		Name:    "general",
		Kind:    "text",
	})
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestListChannels_ZeroActorID_ReturnsErr(t *testing.T) {
	t.Parallel()
	uc := usecase.NewListChannels(newFakeChannelRepo(), newFakeMembershipQuery())
	_, err := uc.Execute(context.Background(), usecase.ListChannelsInput{
		ActorID: uuid.Nil,
		RoomID:  uuid.New(),
	})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestListChannels_ZeroRoomID_ReturnsErr(t *testing.T) {
	t.Parallel()
	uc := usecase.NewListChannels(newFakeChannelRepo(), newFakeMembershipQuery())
	_, err := uc.Execute(context.Background(), usecase.ListChannelsInput{
		ActorID: uuid.New(),
		RoomID:  uuid.Nil,
	})
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestDeleteChannel_ZeroActorID_ReturnsErr(t *testing.T) {
	t.Parallel()
	sut := newDeleteChannelSUT(t)
	err := sut.uc.Execute(context.Background(), usecase.DeleteChannelInput{
		ActorID:   uuid.Nil,
		RoomID:    uuid.New(),
		ChannelID: uuid.New(),
	})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestDeleteChannel_ZeroRoomID_ReturnsErr(t *testing.T) {
	t.Parallel()
	sut := newDeleteChannelSUT(t)
	err := sut.uc.Execute(context.Background(), usecase.DeleteChannelInput{
		ActorID:   uuid.New(),
		RoomID:    uuid.Nil,
		ChannelID: uuid.New(),
	})
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestDeleteChannel_ZeroChannelID_ReturnsErr(t *testing.T) {
	t.Parallel()
	sut := newDeleteChannelSUT(t)
	err := sut.uc.Execute(context.Background(), usecase.DeleteChannelInput{
		ActorID:   uuid.New(),
		RoomID:    uuid.New(),
		ChannelID: uuid.Nil,
	})
	if !errors.Is(err, domain.ErrInvalidChannelID) {
		t.Fatalf("got %v, want ErrInvalidChannelID", err)
	}
}
