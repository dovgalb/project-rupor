package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func TestNewRefreshTokenID_Zero(t *testing.T) {
	t.Parallel()

	_, err := domain.NewRefreshTokenID(uuid.Nil)
	if !errors.Is(err, domain.ErrInvalidRefreshTokenID) {
		t.Fatalf("got %v, want ErrInvalidRefreshTokenID", err)
	}
}
