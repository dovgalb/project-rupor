package middleware_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpxmw "github.com/dovgalb/project-rupor/pkg/httpx/middleware"
)

func TestLogger_LogsAccessFields(t *testing.T) {
	t.Parallel()

	logger, buf := newTestLogger()
	mw := httpxmw.Logger(logger, nil)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/path", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	out := buf.String()
	for _, kw := range []string{`"method":"GET"`, `"path":"/path"`, `"status":200`, `"duration"`, `"request_id"`, `"level":"INFO"`} {
		if !strings.Contains(out, kw) {
			t.Fatalf("лог не содержит %q, out=%s", kw, out)
		}
	}
}

func TestLogger_CapturesResponseStatus(t *testing.T) {
	t.Parallel()

	t.Run("explicit_404", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger()
		mw := httpxmw.Logger(logger, nil)
		h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if !strings.Contains(buf.String(), `"status":404`) {
			t.Fatalf("buf=%s", buf.String())
		}
	})

	t.Run("default_200", func(t *testing.T) {
		t.Parallel()
		logger, buf := newTestLogger()
		mw := httpxmw.Logger(logger, nil)
		h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if !strings.Contains(buf.String(), `"status":200`) {
			t.Fatalf("buf=%s", buf.String())
		}
	})
}

func TestLogger_IncludesRequestID(t *testing.T) {
	t.Parallel()

	logger, buf := newTestLogger()
	c := &counterGen{}
	chain := httpxmw.RequestID(c.gen)(httpxmw.Logger(logger, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	rid := rec.Header().Get("X-Request-ID")
	if rid == "" {
		t.Fatalf("rid пуст")
	}
	if !strings.Contains(buf.String(), `"request_id":"`+rid+`"`) {
		t.Fatalf("лог не содержит rid=%s, buf=%s", rid, buf.String())
	}
}

func TestLogger_HookAddsAttrs(t *testing.T) {
	t.Parallel()

	logger, buf := newTestLogger()
	hook := func(ctx context.Context) []slog.Attr {
		return []slog.Attr{slog.String("user_id", "abc-123")}
	}
	mw := httpxmw.Logger(logger, hook)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !strings.Contains(buf.String(), `"user_id":"abc-123"`) {
		t.Fatalf("buf=%s", buf.String())
	}
}

func TestLogger_DoesNotLogAuthorization(t *testing.T) {
	t.Parallel()

	logger, buf := newTestLogger()
	mw := httpxmw.Logger(logger, nil)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret-token-xxx")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	out := strings.ToLower(buf.String())
	if strings.Contains(out, "secret-token-xxx") {
		t.Fatalf("лог содержит токен, buf=%s", buf.String())
	}
	if strings.Contains(out, `"authorization"`) {
		t.Fatalf("лог содержит ключ authorization, buf=%s", buf.String())
	}
}

func TestLogger_DoesNotLogBody(t *testing.T) {
	t.Parallel()

	logger, buf := newTestLogger()
	mw := httpxmw.Logger(logger, nil)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"password":"hunter2"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if strings.Contains(buf.String(), "hunter2") {
		t.Fatalf("лог содержит body, buf=%s", buf.String())
	}
}
