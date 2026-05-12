//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func dbConn(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func truncate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(
		ctx,
		"TRUNCATE channels, rooms, room_members, invites, refresh_tokens, users CASCADE",
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// seedRoom вставляет пользователя+комнату+owner-membership через прямой SQL.
// Используется в тестах channel-repo, чтобы не зависеть от room/repository.
func seedRoom(t *testing.T, pool *pgxpool.Pool, roomID, ownerID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, username, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		ownerID, ownerID.String()+"@test.local", "hash", "user_"+ownerID.String()[:8], now,
	); err != nil {
		t.Fatalf("seedRoom: insert user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO rooms (id, owner_id, name, created_at) VALUES ($1, $2, $3, $4)`,
		roomID, ownerID, "test-room", now,
	); err != nil {
		t.Fatalf("seedRoom: insert room: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO room_members (room_id, user_id, role, joined_at) VALUES ($1, $2, 'owner', $3)`,
		roomID, ownerID, now,
	); err != nil {
		t.Fatalf("seedRoom: insert membership: %v", err)
	}
}
