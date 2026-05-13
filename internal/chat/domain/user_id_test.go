package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
)

func TestNewUserID_Valid_Constructs(t *testing.T) {
	t.Parallel()

	raw := uuid.New()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.UUID() != raw {
		t.Fatalf("UUID() = %v, want %v", id.UUID(), raw)
	}
	if id.String() != raw.String() {
		t.Fatalf("String() = %q, want %q", id.String(), raw.String())
	}
	if id.IsZero() {
		t.Fatal("IsZero() = true, want false")
	}
}

func TestNewUserID_Zero_ReturnsErr(t *testing.T) {
	t.Parallel()

	_, err := domain.NewUserID(uuid.Nil)
	if !errors.Is(err, domain.ErrInvalidAuthorID) {
		t.Fatalf("got %v, want ErrInvalidAuthorID", err)
	}
}

func TestUserID_ZeroValue_IsZero(t *testing.T) {
	t.Parallel()

	var id domain.UserID
	if !id.IsZero() {
		t.Fatal("zero value: IsZero() = false, want true")
	}
}
