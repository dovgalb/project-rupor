package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func TestNewUserID_Zero(t *testing.T) {
	t.Parallel()

	_, err := domain.NewUserID(uuid.Nil)
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}
