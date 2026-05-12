package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

func TestNewChannelID_Valid_Constructs(t *testing.T) {
	t.Parallel()

	raw := uuid.New()
	id, err := domain.NewChannelID(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.UUID() != raw {
		t.Fatalf("UUID() = %v, want %v", id.UUID(), raw)
	}
}

func TestNewChannelID_Zero_ReturnsErr(t *testing.T) {
	t.Parallel()

	_, err := domain.NewChannelID(uuid.Nil)
	if !errors.Is(err, domain.ErrInvalidChannelID) {
		t.Fatalf("got %v, want ErrInvalidChannelID", err)
	}
}
