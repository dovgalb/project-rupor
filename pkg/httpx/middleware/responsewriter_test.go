package middleware

import (
	"bufio"
	"net"
	"net/http"
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

func TestResponseWriter_HijackProxies(t *testing.T) {
	t.Parallel()

	// Имитируем underlying ResponseWriter, реализующий Hijacker.
	hijackable := &fakeHijackable{}
	sw := wrap(hijackable)

	conn, rw, err := sw.Hijack()
	if err != nil {
		t.Fatalf("Hijack: %v", err)
	}
	if conn != nil || rw != nil {
		t.Fatalf("ожидали nil-conn от fake")
	}
	if !hijackable.called {
		t.Fatalf("underlying Hijack не вызван")
	}
	if sw.status != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", sw.status)
	}
}

func TestResponseWriter_HijackFails_WhenUnderlyingDoesNotSupport(t *testing.T) {
	t.Parallel()

	sw := wrap(httptest.NewRecorder()) // httptest.ResponseRecorder НЕ реализует Hijacker
	_, _, err := sw.Hijack()
	if err == nil {
		t.Fatalf("ожидали ошибку, получили nil")
	}
}

func TestResponseWriter_FlushProxies(t *testing.T) {
	t.Parallel()

	flushable := &fakeFlushable{}
	sw := wrap(flushable)
	sw.Flush()
	if !flushable.called {
		t.Fatalf("underlying Flush не вызван")
	}
}

// --- fakes для проверки type-assert'ов ---

type fakeHijackable struct {
	httptest.ResponseRecorder
	called bool
}

func (f *fakeHijackable) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	f.called = true
	return nil, nil, nil
}

type fakeFlushable struct {
	httptest.ResponseRecorder
	called bool
}

func (f *fakeFlushable) Flush() {
	f.called = true
}
