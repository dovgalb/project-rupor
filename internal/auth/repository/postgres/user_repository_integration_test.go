//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres/db"
)

func TestUserRepository_Save_RoundTrip(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")

	pool := dbConn(t)
	defer truncate(t, pool, "refresh_tokens", "users")

	ctx := context.Background()
	repo := postgres.NewUserRepository(db.New(pool))

	id, _ := domain.NewUserID(uuid.New())
	email, _ := domain.NewEmail("user@example.com")
	username, _ := domain.NewUsername("user_1")
	hash, _ := domain.NewPasswordHash("$2a$10$hash")
	u, err := domain.NewUser(id, email, username, hash, time.Now().UTC())
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}

	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ID() != u.ID() || got.Email() != u.Email() || got.Username() != u.Username() {
		t.Fatalf("round-trip mismatch")
	}
}

func TestUserRepository_Save_DuplicateEmail(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = domain.ErrEmailAlreadyTaken
	_ = errors.Is
}

func TestUserRepository_Save_DuplicateUsername(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = domain.ErrUsernameAlreadyTaken
}

func TestUserRepository_FindByEmail_CaseInsensitive(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = domain.ErrUserNotFound
}
