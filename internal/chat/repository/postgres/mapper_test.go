package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
	"github.com/dovgalb/project-rupor/internal/chat/repository/postgres/db"
)

func mustMessageID(t *testing.T, raw uuid.UUID) domain.MessageID {
	t.Helper()
	id, err := domain.NewMessageID(raw)
	if err != nil {
		t.Fatalf("NewMessageID: %v", err)
	}
	return id
}

func mustChannelID(t *testing.T, raw uuid.UUID) domain.ChannelID {
	t.Helper()
	id, err := domain.NewChannelID(raw)
	if err != nil {
		t.Fatalf("NewChannelID: %v", err)
	}
	return id
}

func mustUserID(t *testing.T, raw uuid.UUID) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("NewUserID: %v", err)
	}
	return id
}

func mustMessageText(t *testing.T, raw string) domain.MessageText {
	t.Helper()
	v, err := domain.NewMessageText(raw)
	if err != nil {
		t.Fatalf("NewMessageText: %v", err)
	}
	return v
}

func TestMessageMapper_RoundTrip(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	src, err := domain.NewMessage(
		mustMessageID(t, uuid.New()),
		mustChannelID(t, uuid.New()),
		mustUserID(t, uuid.New()),
		mustMessageText(t, "hello"),
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}

	params := domainToInsertMessageParams(src)
	row := db.Message(params)

	got, err := messageRowToDomain(row)
	if err != nil {
		t.Fatalf("messageRowToDomain: %v", err)
	}
	if got.ID() != src.ID() {
		t.Fatalf("ID mismatch")
	}
	if got.ChannelID() != src.ChannelID() {
		t.Fatalf("ChannelID mismatch")
	}
	if got.AuthorID() != src.AuthorID() {
		t.Fatalf("AuthorID mismatch")
	}
	if got.Text() != src.Text() {
		t.Fatalf("Text mismatch")
	}
	if !got.CreatedAt().Equal(src.CreatedAt()) {
		t.Fatalf("CreatedAt mismatch")
	}
}

func TestOptionalUUID_Zero_ReturnsInvalid(t *testing.T) {
	t.Parallel()

	got := optionalUUID(domain.MessageID{})
	if got.Valid {
		t.Fatal("zero MessageID: Valid = true, want false")
	}
}

func TestOptionalUUID_NonZero_ReturnsValid(t *testing.T) {
	t.Parallel()

	raw := uuid.New()
	id := mustMessageID(t, raw)
	got := optionalUUID(id)
	if !got.Valid {
		t.Fatal("non-zero MessageID: Valid = false, want true")
	}
	if got.Bytes != raw {
		t.Fatalf("Bytes = %v, want %v", got.Bytes, raw)
	}
}
