package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
)

func TestNewMessage_Valid_ReturnsMessage(t *testing.T) {
	t.Parallel()

	id := mustMessageID(t, uuid.New())
	channelID := mustChannelID(t, uuid.New())
	authorID := mustUserID(t, uuid.New())
	text := mustMessageText(t, "hello world")

	m, err := domain.NewMessage(id, channelID, authorID, text, defaultBuilderTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID() != id {
		t.Fatalf("ID mismatch")
	}
	if m.ChannelID() != channelID {
		t.Fatalf("ChannelID mismatch")
	}
	if m.AuthorID() != authorID {
		t.Fatalf("AuthorID mismatch")
	}
	if m.Text() != text {
		t.Fatalf("Text mismatch")
	}
	if !m.CreatedAt().Equal(defaultBuilderTime) {
		t.Fatalf("CreatedAt mismatch")
	}
}

func TestNewMessage_ZeroID_ReturnsErrInvalidMessageID(t *testing.T) {
	t.Parallel()

	_, err := domain.NewMessage(
		domain.MessageID{},
		mustChannelID(t, uuid.New()),
		mustUserID(t, uuid.New()),
		mustMessageText(t, "hello"),
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidMessageID) {
		t.Fatalf("got %v, want ErrInvalidMessageID", err)
	}
}

func TestNewMessage_ZeroChannelID_ReturnsErrInvalidChannelID(t *testing.T) {
	t.Parallel()

	_, err := domain.NewMessage(
		mustMessageID(t, uuid.New()),
		domain.ChannelID{},
		mustUserID(t, uuid.New()),
		mustMessageText(t, "hello"),
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidChannelID) {
		t.Fatalf("got %v, want ErrInvalidChannelID", err)
	}
}

func TestNewMessage_ZeroAuthorID_ReturnsErrInvalidAuthorID(t *testing.T) {
	t.Parallel()

	_, err := domain.NewMessage(
		mustMessageID(t, uuid.New()),
		mustChannelID(t, uuid.New()),
		domain.UserID{},
		mustMessageText(t, "hello"),
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidAuthorID) {
		t.Fatalf("got %v, want ErrInvalidAuthorID", err)
	}
}

func TestNewMessage_ZeroCreatedAt_ReturnsErrInvalidCreatedAt(t *testing.T) {
	t.Parallel()

	_, err := domain.NewMessage(
		mustMessageID(t, uuid.New()),
		mustChannelID(t, uuid.New()),
		mustUserID(t, uuid.New()),
		mustMessageText(t, "hello"),
		time.Time{},
	)
	if !errors.Is(err, domain.ErrInvalidCreatedAt) {
		t.Fatalf("got %v, want ErrInvalidCreatedAt", err)
	}
}

func TestReconstructMessage_Valid_Constructs(t *testing.T) {
	t.Parallel()

	id := mustMessageID(t, uuid.New())
	channelID := mustChannelID(t, uuid.New())
	authorID := mustUserID(t, uuid.New())
	text := mustMessageText(t, "reconstructed")

	m, err := domain.ReconstructMessage(id, channelID, authorID, text, defaultBuilderTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Text() != text {
		t.Fatalf("Text mismatch")
	}
}
