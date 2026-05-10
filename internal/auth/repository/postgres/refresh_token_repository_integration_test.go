//go:build integration

package postgres_test

import (
	"testing"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres"
)

func TestRefreshTokenRepository_Save_RoundTrip(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = postgres.NewRefreshTokenRepository
}

func TestRefreshTokenRepository_FindByHash_NotFound(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = domain.ErrRefreshTokenNotFound
}

func TestRefreshTokenRepository_Rotate_Atomic(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
}

func TestRefreshTokenRepository_Rotate_AlreadyRevoked(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = domain.ErrRefreshTokenRevoked
}
