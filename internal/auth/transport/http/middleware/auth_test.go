package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	jwtv5 "github.com/golang-jwt/jwt/v5"

	jwtadapter "github.com/dovgalb/project-rupor/internal/auth/repository/jwt"
	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

const testSecret = "test-secret-1234567890"

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func newAuthRouter(t *testing.T, issuer usecase.TokenIssuer, clock usecase.Clock) (chi.Router, *bool) {
	t.Helper()
	called := false
	r := chi.NewRouter()
	r.Use(authmw.RequireAuth(issuer, clock))
	r.Get("/protected", func(w http.ResponseWriter, req *http.Request) {
		uid, ok := authmw.UserIDFromContext(req.Context())
		if !ok {
			t.Errorf("UserIDFromContext: ok = false")
		}
		called = true
		_, _ = fmt.Fprintf(w, "ok: %s", uid.String())
	})
	return r, &called
}

func defaultClock() fixedClock {
	return fixedClock{now: time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)}
}

func TestRequireAuth_Success(t *testing.T) {
	t.Parallel()

	clock := defaultClock()
	issuer := jwtadapter.NewTokenIssuer([]byte(testSecret), 15*time.Minute)
	uid := mustUserID(t, "550e8400-e29b-41d4-a716-446655440000")
	tok, _, err := issuer.IssueAccess(uid, clock.Now())
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}

	r, called := newAuthRouter(t, issuer, clock)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Code = %d, body=%s", rec.Code, rec.Body.String())
	}
	if !*called {
		t.Fatalf("handler не вызван")
	}
	if !strings.Contains(rec.Body.String(), "ok: "+uid.String()) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestRequireAuth_NoHeader(t *testing.T) {
	t.Parallel()

	clock := defaultClock()
	issuer := jwtadapter.NewTokenIssuer([]byte(testSecret), 15*time.Minute)
	r, called := newAuthRouter(t, issuer, clock)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"AUTH-010"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if *called {
		t.Fatalf("handler вызван при ошибке")
	}
}

func TestRequireAuth_BadScheme(t *testing.T) {
	t.Parallel()

	clock := defaultClock()
	issuer := jwtadapter.NewTokenIssuer([]byte(testSecret), 15*time.Minute)
	r, called := newAuthRouter(t, issuer, clock)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dGVzdA==")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"AUTH-010"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if *called {
		t.Fatalf("handler вызван")
	}
}

func TestRequireAuth_EmptyToken(t *testing.T) {
	t.Parallel()

	clock := defaultClock()
	issuer := jwtadapter.NewTokenIssuer([]byte(testSecret), 15*time.Minute)
	r, called := newAuthRouter(t, issuer, clock)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"AUTH-010"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if *called {
		t.Fatalf("handler вызван")
	}
}

func TestRequireAuth_InvalidSignature(t *testing.T) {
	t.Parallel()

	clock := defaultClock()
	issuer := jwtadapter.NewTokenIssuer([]byte(testSecret), 15*time.Minute)
	wrongIssuer := jwtadapter.NewTokenIssuer([]byte("wrong-secret-9876543210"), 15*time.Minute)
	uid := mustUserID(t, "550e8400-e29b-41d4-a716-446655440000")
	tok, _, err := wrongIssuer.IssueAccess(uid, clock.Now())
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}

	r, called := newAuthRouter(t, issuer, clock)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"AUTH-010"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if *called {
		t.Fatalf("handler вызван")
	}
}

func TestRequireAuth_TokenExpired(t *testing.T) {
	t.Parallel()

	clock := defaultClock()
	now := clock.Now()
	uid := mustUserID(t, "550e8400-e29b-41d4-a716-446655440000")
	claims := jwtv5.RegisteredClaims{
		Subject:   uid.String(),
		IssuedAt:  jwtv5.NewNumericDate(now.Add(-2 * time.Hour)),
		ExpiresAt: jwtv5.NewNumericDate(now.Add(-1 * time.Hour)),
	}
	tok, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	issuer := jwtadapter.NewTokenIssuer([]byte(testSecret), 15*time.Minute)
	r, called := newAuthRouter(t, issuer, clock)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"AUTH-011"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if *called {
		t.Fatalf("handler вызван")
	}
}

func TestRequireAuth_BadSubClaim(t *testing.T) {
	t.Parallel()

	clock := defaultClock()
	now := clock.Now()
	claims := jwtv5.RegisteredClaims{
		Subject:   "not-a-uuid",
		IssuedAt:  jwtv5.NewNumericDate(now),
		ExpiresAt: jwtv5.NewNumericDate(now.Add(time.Hour)),
	}
	tok, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	issuer := jwtadapter.NewTokenIssuer([]byte(testSecret), 15*time.Minute)
	r, called := newAuthRouter(t, issuer, clock)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"AUTH-010"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if *called {
		t.Fatalf("handler вызван")
	}
}

func TestRequireAuth_NextNotCalledOnError(t *testing.T) {
	t.Parallel()

	clock := defaultClock()
	issuer := jwtadapter.NewTokenIssuer([]byte(testSecret), 15*time.Minute)

	cases := []struct {
		name   string
		header string
	}{
		{"no_header", ""},
		{"basic_scheme", "Basic dGVzdA=="},
		{"empty_token", "Bearer "},
		{"garbage", "Bearer garbage.token.here"},
		{"lowercase_bearer", "bearer something"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r, called := newAuthRouter(t, issuer, clock)
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if *called {
				t.Fatalf("handler вызван при ошибке (%s)", tc.name)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("Code = %d", rec.Code)
			}
		})
	}
}
