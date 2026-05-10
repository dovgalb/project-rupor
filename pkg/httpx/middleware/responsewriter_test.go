package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestResponseWriter_CapturesStatus(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	sw := wrap(rec)
	sw.WriteHeader(404)

	if sw.status != 404 {
		t.Fatalf("status = %d, want 404", sw.status)
	}
	if !sw.headerWritten {
		t.Fatalf("headerWritten = false, want true")
	}
	if rec.Code != 404 {
		t.Fatalf("rec.Code = %d, want 404", rec.Code)
	}
}

func TestResponseWriter_DoubleWriteHeaderIsNoOp(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	sw := wrap(rec)
	sw.WriteHeader(404)
	sw.WriteHeader(500)

	if sw.status != 404 {
		t.Fatalf("status = %d, want 404 (first wins)", sw.status)
	}
	if rec.Code != 404 {
		t.Fatalf("rec.Code = %d, want 404", rec.Code)
	}
}

func TestResponseWriter_WrapIdempotent(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	sw1 := wrap(rec)
	sw2 := wrap(sw1)
	if sw1 != sw2 {
		t.Fatalf("wrap not idempotent: sw1 != sw2")
	}
}
