package domain_test

import (
	"errors"
	"testing"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func TestNewPasswordHash_Empty(t *testing.T) {
	t.Parallel()

	_, err := domain.NewPasswordHash("")
	if !errors.Is(err, domain.ErrInvalidPasswordHash) {
		t.Fatalf("got %v, want ErrInvalidPasswordHash", err)
	}
}
