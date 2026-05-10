package middleware_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
)

func mustUserID(t *testing.T, raw string) domain.UserID {
	t.Helper()
	parsed, err := uuid.Parse(raw)
	if err != nil {
		t.Fatalf("uuid.Parse: %v", err)
	}
	uid, err := domain.NewUserID(parsed)
	if err != nil {
		t.Fatalf("domain.NewUserID: %v", err)
	}
	return uid
}

func TestUserIDContext_RoundTrip(t *testing.T) {
	t.Parallel()

	uid := mustUserID(t, "550e8400-e29b-41d4-a716-446655440000")
	ctx := authmw.WithUserID(context.Background(), uid)
	got, ok := authmw.UserIDFromContext(ctx)
	if !ok {
		t.Fatalf("ok = false")
	}
	if got.String() != uid.String() {
		t.Fatalf("got %q, want %q", got.String(), uid.String())
	}
}

func TestUserIDContext_AbsentReturnsFalse(t *testing.T) {
	t.Parallel()

	_, ok := authmw.UserIDFromContext(context.Background())
	if ok {
		t.Fatalf("ok = true, want false")
	}
}
