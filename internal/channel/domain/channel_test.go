package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

func TestNewChannel_Valid_Constructs(t *testing.T) {
	t.Parallel()

	id := mustChannelID(t, uuid.New())
	roomID := mustRoomID(t, uuid.New())
	name := mustChannelName(t, "general")

	c, err := domain.NewChannel(id, roomID, name, domain.ChannelKindText, defaultBuilderTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID() != id {
		t.Fatalf("ID mismatch")
	}
	if c.RoomID() != roomID {
		t.Fatalf("RoomID mismatch")
	}
	if c.Name() != name {
		t.Fatalf("Name mismatch")
	}
	if c.Kind() != domain.ChannelKindText {
		t.Fatalf("Kind mismatch")
	}
	if !c.CreatedAt().Equal(defaultBuilderTime) {
		t.Fatalf("CreatedAt mismatch")
	}
}

func TestNewChannel_ZeroID_ReturnsErr(t *testing.T) {
	t.Parallel()

	_, err := domain.NewChannel(
		domain.ChannelID{},
		mustRoomID(t, uuid.New()),
		mustChannelName(t, "general"),
		domain.ChannelKindText,
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidChannelID) {
		t.Fatalf("got %v, want ErrInvalidChannelID", err)
	}
}

func TestNewChannel_ZeroRoomID_ReturnsErr(t *testing.T) {
	t.Parallel()

	_, err := domain.NewChannel(
		mustChannelID(t, uuid.New()),
		domain.RoomID{},
		mustChannelName(t, "general"),
		domain.ChannelKindText,
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestNewChannel_InvalidKind_ReturnsErr(t *testing.T) {
	t.Parallel()

	_, err := domain.NewChannel(
		mustChannelID(t, uuid.New()),
		mustRoomID(t, uuid.New()),
		mustChannelName(t, "general"),
		domain.ChannelKind(99),
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidChannelKind) {
		t.Fatalf("got %v, want ErrInvalidChannelKind", err)
	}
}

func TestNewChannel_ZeroCreatedAt_ReturnsErr(t *testing.T) {
	t.Parallel()

	_, err := domain.NewChannel(
		mustChannelID(t, uuid.New()),
		mustRoomID(t, uuid.New()),
		mustChannelName(t, "general"),
		domain.ChannelKindText,
		time.Time{},
	)
	if !errors.Is(err, domain.ErrInvalidCreatedAt) {
		t.Fatalf("got %v, want ErrInvalidCreatedAt", err)
	}
}

func TestReconstructChannel_Valid_Constructs(t *testing.T) {
	t.Parallel()

	id := mustChannelID(t, uuid.New())
	roomID := mustRoomID(t, uuid.New())
	name := mustChannelName(t, "voice-room")

	c, err := domain.ReconstructChannel(id, roomID, name, domain.ChannelKindVoice, defaultBuilderTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Kind() != domain.ChannelKindVoice {
		t.Fatalf("Kind mismatch")
	}
}
