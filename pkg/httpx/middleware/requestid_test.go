package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dovgalb/project-rupor/pkg/httpx"
	httpxmw "github.com/dovgalb/project-rupor/pkg/httpx/middleware"
)

type counterGen struct{ n int }

func (c *counterGen) gen() string {
	c.n++
	return "generated-" + itoa(c.n)
}

func itoa(n int) string {
	// малая утилита, чтобы не тащить strconv ради одной цифры в тестах
	if n == 0 {
		return "0"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

func newRouterWithRequestID(gen func() string, capture *string) http.Handler {
	mw := httpxmw.RequestID(gen)
	return mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid, _ := httpx.RequestIDFromContext(r.Context())
		if capture != nil {
			*capture = rid
		}
		w.WriteHeader(http.StatusOK)
	}))
}

func TestRequestID_GeneratesWhenAbsent(t *testing.T) {
	t.Parallel()

	c := &counterGen{}
	var ctxRID string
	h := newRouterWithRequestID(c.gen, &ctxRID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if c.n != 1 {
		t.Fatalf("uuidGen calls = %d, want 1", c.n)
	}
	got := rec.Header().Get("X-Request-ID")
	if got != "generated-1" {
		t.Fatalf("response X-Request-ID = %q", got)
	}
	if ctxRID != got {
		t.Fatalf("ctx rid %q != response %q", ctxRID, got)
	}
}

func TestRequestID_AcceptsValidIncoming(t *testing.T) {
	t.Parallel()

	c := &counterGen{}
	h := newRouterWithRequestID(c.gen, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "my-correlation-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if c.n != 0 {
		t.Fatalf("uuidGen called %d times, want 0", c.n)
	}
	if got := rec.Header().Get("X-Request-ID"); got != "my-correlation-1" {
		t.Fatalf("X-Request-ID = %q", got)
	}
}

func TestRequestID_RejectsInvalidIncoming(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
	}{
		{"too_long", strings.Repeat("a", 129)},
		{"crlf", "bad\r\nX-Hack: 1"},
		{"null_byte", "bad\x00rid"},
		{"non_ascii", "τεστ"},
		{"empty", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := &counterGen{}
			h := newRouterWithRequestID(c.gen, nil)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.value != "" {
				req.Header.Set("X-Request-ID", tc.value)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if c.n != 1 {
				t.Fatalf("uuidGen calls = %d, want 1", c.n)
			}
			if got := rec.Header().Get("X-Request-ID"); got == tc.value && tc.value != "" {
				t.Fatalf("middleware accepted invalid value %q", tc.value)
			}
		})
	}
}

func TestRequestID_SetsResponseHeader(t *testing.T) {
	t.Parallel()

	c := &counterGen{}
	mw := httpxmw.RequestID(c.gen)
	var headerAtHandler string
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerAtHandler = w.Header().Get("X-Request-ID")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if headerAtHandler == "" {
		t.Fatalf("X-Request-ID не выставлен ДО handler-а")
	}
	if rec.Header().Get("X-Request-ID") != headerAtHandler {
		t.Fatalf("response header не совпадает с пред-handler значением")
	}
}

func TestRequestID_ProvidesContextValue(t *testing.T) {
	t.Parallel()

	c := &counterGen{}
	var ctxRID string
	h := newRouterWithRequestID(c.gen, &ctxRID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if ctxRID == "" {
		t.Fatalf("ctx rid пуст")
	}
	if rec.Header().Get("X-Request-ID") != ctxRID {
		t.Fatalf("response header не совпадает с ctx")
	}
}
