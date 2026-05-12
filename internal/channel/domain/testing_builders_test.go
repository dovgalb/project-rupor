package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

var defaultBuilderTime = time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)

func mustChannelID(t *testing.T, raw uuid.UUID) domain.ChannelID {
	t.Helper()
	id, err := domain.NewChannelID(raw)
	if err != nil {
		t.Fatalf("mustChannelID(%s): %v", raw, err)
	}
	return id
}

func mustRoomID(t *testing.T, raw uuid.UUID) domain.RoomID {
	t.Helper()
	id, err := domain.NewRoomID(raw)
	if err != nil {
		t.Fatalf("mustRoomID(%s): %v", raw, err)
	}
	return id
}

func mustChannelName(t *testing.T, raw string) domain.ChannelName {
	t.Helper()
	n, err := domain.NewChannelName(raw)
	if err != nil {
		t.Fatalf("mustChannelName(%q): %v", raw, err)
	}
	return n
}

type ChannelBuilder struct {
	t         *testing.T
	id        uuid.UUID
	roomID    uuid.UUID
	name      string
	kind      domain.ChannelKind
	createdAt time.Time
}

func NewChannelBuilder(t *testing.T) *ChannelBuilder {
	t.Helper()
	return &ChannelBuilder{
		t:         t,
		id:        uuid.New(),
		roomID:    uuid.New(),
		name:      "general",
		kind:      domain.ChannelKindText,
		createdAt: defaultBuilderTime,
	}
}

func (b *ChannelBuilder) WithID(id uuid.UUID) *ChannelBuilder   { b.id = id; return b }
func (b *ChannelBuilder) WithRoom(id uuid.UUID) *ChannelBuilder { b.roomID = id; return b }
func (b *ChannelBuilder) WithName(name string) *ChannelBuilder  { b.name = name; return b }
func (b *ChannelBuilder) WithKind(k domain.ChannelKind) *ChannelBuilder {
	b.kind = k
	return b
}
func (b *ChannelBuilder) Text() *ChannelBuilder                 { b.kind = domain.ChannelKindText; return b }
func (b *ChannelBuilder) Voice() *ChannelBuilder                { b.kind = domain.ChannelKindVoice; return b }
func (b *ChannelBuilder) CreatedAt(t time.Time) *ChannelBuilder { b.createdAt = t; return b }

func (b *ChannelBuilder) Build() *domain.Channel {
	b.t.Helper()
	c, err := domain.NewChannel(
		mustChannelID(b.t, b.id),
		mustRoomID(b.t, b.roomID),
		mustChannelName(b.t, b.name),
		b.kind,
		b.createdAt,
	)
	if err != nil {
		b.t.Fatalf("ChannelBuilder.Build: %v", err)
	}
	return c
}
