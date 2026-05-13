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
	if _, err := pool.Exec(
		context.Background(),
		"TRUNCATE messages, channels, room_members, invites, rooms, refresh_tokens, users CASCADE",
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// seedChannel создаёт user → room → owner-membership → channel.
func seedChannel(t *testing.T, pool *pgxpool.Pool, channelID, roomID, ownerID uuid.UUID, kind string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, username, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		ownerID, ownerID.String()+"@test.local", "hash", "user_"+ownerID.String()[:8], now,
	); err != nil {
		t.Fatalf("seedChannel: insert user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO rooms (id, owner_id, name, created_at) VALUES ($1, $2, $3, $4)`,
		roomID, ownerID, "test-room", now,
	); err != nil {
		t.Fatalf("seedChannel: insert room: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO room_members (room_id, user_id, role, joined_at) VALUES ($1, $2, 'owner', $3)`,
		roomID, ownerID, now,
	); err != nil {
		t.Fatalf("seedChannel: insert membership: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO channels (id, room_id, name, kind, created_at) VALUES ($1, $2, $3, $4, $5)`,
		channelID, roomID, "general", kind, now,
	); err != nil {
		t.Fatalf("seedChannel: insert channel: %v", err)
	}
}
