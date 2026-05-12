package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
)

var defaultBuilderTime = time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)

func mustMessageID(t *testing.T, raw uuid.UUID) domain.MessageID {
	t.Helper()
	id, err := domain.NewMessageID(raw)
	if err != nil {
		t.Fatalf("mustMessageID(%s): %v", raw, err)
	}
	return id
}

func mustChannelID(t *testing.T, raw uuid.UUID) domain.ChannelID {
	t.Helper()
	id, err := domain.NewChannelID(raw)
	if err != nil {
		t.Fatalf("mustChannelID(%s): %v", raw, err)
	}
	return id
}

func mustUserID(t *testing.T, raw uuid.UUID) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("mustUserID(%s): %v", raw, err)
	}
	return id
}

func mustMessageText(t *testing.T, raw string) domain.MessageText {
	t.Helper()
	v, err := domain.NewMessageText(raw)
	if err != nil {
		t.Fatalf("mustMessageText(%q): %v", raw, err)
	}
	return v
}

type MessageBuilder struct {
	t         *testing.T
	id        uuid.UUID
	channelID uuid.UUID
	authorID  uuid.UUID
	text      string
	createdAt time.Time
}

func NewMessageBuilder(t *testing.T) *MessageBuilder {
	t.Helper()
	return &MessageBuilder{
		t:         t,
		id:        uuid.New(),
		channelID: uuid.New(),
		authorID:  uuid.New(),
		text:      "hello",
		createdAt: defaultBuilderTime,
	}
}

func (b *MessageBuilder) WithID(id uuid.UUID) *MessageBuilder        { b.id = id; return b }
func (b *MessageBuilder) WithChannelID(id uuid.UUID) *MessageBuilder { b.channelID = id; return b }
func (b *MessageBuilder) WithAuthorID(id uuid.UUID) *MessageBuilder  { b.authorID = id; return b }
func (b *MessageBuilder) WithText(s string) *MessageBuilder          { b.text = s; return b }
func (b *MessageBuilder) WithCreatedAt(t time.Time) *MessageBuilder  { b.createdAt = t; return b }

func (b *MessageBuilder) Build() *domain.Message {
	b.t.Helper()
	m, err := domain.NewMessage(
		mustMessageID(b.t, b.id),
		mustChannelID(b.t, b.channelID),
		mustUserID(b.t, b.authorID),
		mustMessageText(b.t, b.text),
		b.createdAt,
	)
	if err != nil {
		b.t.Fatalf("MessageBuilder.Build: %v", err)
	}
	return m
}
