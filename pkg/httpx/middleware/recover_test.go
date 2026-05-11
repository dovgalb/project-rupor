package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpxmw "github.com/dovgalb/project-rupor/pkg/httpx/middleware"
)

func newTestLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return logger, buf
}

func TestRecover_PanicReturnsInternalEnvelope(t *testing.T) {
	t.Parallel()

	logger, _ := newTestLogger()
	mw := httpxmw.Recover(logger)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("forced for test")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Code = %d, want 500", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"code":"INTERNAL"`) {
		t.Fatalf("body = %s", body)
	}
	if !strings.Contains(body, `"message":"internal"`) {
		t.Fatalf("body = %s", body)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestRecover_PassesAbortHandler(t *testing.T) {
	t.Parallel()

	logger, _ := newTestLogger()
	mw := httpxmw.Recover(logger)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		v := recover()
		if v == nil {
			t.Fatalf("ожидался re-panic с http.ErrAbortHandler")
		}
		if v != http.ErrAbortHandler {
			t.Fatalf("re-panic value = %v", v)
		}
	}()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
}

func TestRecover_PanicAfterWriteHeader_LogsAndContinues(t *testing.T) {
	t.Parallel()

	logger, buf := newTestLogger()
	mw := httpxmw.Recover(logger)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		panic("oops")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Code = %d, want 200 (не должен перезаписаться)", rec.Code)
	}
	if !strings.Contains(buf.String(), "panic after WriteHeader") {
		t.Fatalf("лог не содержит warning, buf=%s", buf.String())
	}
}

func TestRecover_LogsStackAndRequestID(t *testing.T) {
	t.Parallel()

	logger, buf := newTestLogger()
	c := &counterGen{}
	chain := httpxmw.RequestID(c.gen)(httpxmw.Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})))

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	out := buf.String()
	for _, kw := range []string{"panic recovered", `"err"`, `"stack"`, `"request_id"`, `"method"`, `"path"`} {
		if !strings.Contains(out, kw) {
			t.Fatalf("лог не содержит %q, out=%s", kw, out)
		}
	}
}
