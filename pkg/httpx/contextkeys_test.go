package httpx_test

import (
	"context"
	"testing"

	"github.com/dovgalb/project-rupor/pkg/httpx"
)

func TestRequestIDContext_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := httpx.WithRequestID(context.Background(), "rid-1")
	got, ok := httpx.RequestIDFromContext(ctx)
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if got != "rid-1" {
		t.Fatalf("got = %q, want rid-1", got)
	}
}

func TestRequestIDContext_AbsentReturnsEmpty(t *testing.T) {
	t.Parallel()

	got, ok := httpx.RequestIDFromContext(context.Background())
	if ok {
		t.Fatalf("ok = true, want false")
	}
	if got != "" {
		t.Fatalf("got = %q, want empty", got)
	}
}
