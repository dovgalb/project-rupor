package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpxmw "github.com/dovgalb/project-rupor/pkg/httpx/middleware"
)

const allowedOrigin = "http://localhost:5173"

func newCORSChain(allow []string, downstream http.HandlerFunc) http.Handler {
	mw := httpxmw.CORS(allow, false)
	return mw(downstream)
}

func TestCORS_PreflightAllowedOrigin(t *testing.T) {
	t.Parallel()

	h := newCORSChain([]string{allowedOrigin}, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("downstream не должен вызываться на preflight")
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", allowedOrigin)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("Code = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Fatalf("Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q", got)
	}
}

func TestCORS_PreflightDeniedOrigin(t *testing.T) {
	t.Parallel()

	h := newCORSChain([]string{allowedOrigin}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Code = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty", got)
	}
}

func TestCORS_NonOptionsAllowedOrigin(t *testing.T) {
	t.Parallel()

	called := false
	h := newCORSChain([]string{allowedOrigin}, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", allowedOrigin)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("downstream не вызван")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Fatalf("Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q", got)
	}
}

func TestCORS_NonOptionsDeniedOrigin(t *testing.T) {
	t.Parallel()

	called := false
	h := newCORSChain([]string{allowedOrigin}, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("downstream должен быть вызван")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty", got)
	}
}

func TestCORS_OptionsWithoutOrigin(t *testing.T) {
	t.Parallel()

	called := false
	h := newCORSChain([]string{allowedOrigin}, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("downstream должен быть вызван (это not-CORS запрос)")
	}
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Code = %d", rec.Code)
	}
}

func TestCORS_PreflightHeaders(t *testing.T) {
	t.Parallel()

	h := newCORSChain([]string{allowedOrigin}, func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", allowedOrigin)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, PUT, DELETE, OPTIONS" {
		t.Fatalf("Allow-Methods = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type, X-Request-ID" {
		t.Fatalf("Allow-Headers = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "600" {
		t.Fatalf("Max-Age = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Allow-Credentials = %q, want empty", got)
	}
}

func TestCORS_WildcardNotSupported(t *testing.T) {
	t.Parallel()

	h := newCORSChain([]string{"*"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://anywhere.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty (wildcard не интерпретируется)", got)
	}
}
