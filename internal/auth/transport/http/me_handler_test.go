package httpauth_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func getMe(t *testing.T, env *testEnv, authHeader string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, env.server.URL+"/auth/me", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func TestMeHandler_Success(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	registerUser(t, env, "user@example.com", "user_1", "correct-pass")
	access, _ := loginAndGetTokens(t, env, "user@example.com", "correct-pass")

	resp := getMe(t, env, "Bearer "+access)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, err := uuid.Parse(body.ID); err != nil {
		t.Fatalf("ID is not UUID: %q", body.ID)
	}
	if body.Email != "user@example.com" || body.Username != "user_1" {
		t.Fatalf("body mismatch: %+v", body)
	}
}

func TestMeHandler_NoAuth(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	resp := getMe(t, env, "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "AUTH-010" {
		t.Fatalf("code = %q, want AUTH-010", er.Error.Code)
	}
}

func TestMeHandler_BadScheme(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	resp := getMe(t, env, "Basic dXNlcjpwYXNz")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "AUTH-010" {
		t.Fatalf("code = %q, want AUTH-010", er.Error.Code)
	}
}

func TestMeHandler_InvalidSignature(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	registerUser(t, env, "user@example.com", "user_1", "correct-pass")
	access, _ := loginAndGetTokens(t, env, "user@example.com", "correct-pass")

	parts := strings.Split(access, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token shape")
	}
	tampered := parts[0] + "." + parts[1] + "." + "AAAA" + parts[2]

	resp := getMe(t, env, "Bearer "+tampered)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "AUTH-010" {
		t.Fatalf("code = %q, want AUTH-010", er.Error.Code)
	}
}

func TestMeHandler_TokenExpired(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	uid := uuid.New()
	claims := jwtv5.RegisteredClaims{
		Subject:   uid.String(),
		IssuedAt:  jwtv5.NewNumericDate(env.clock.now.Add(-2 * testAccessTTL)),
		ExpiresAt: jwtv5.NewNumericDate(env.clock.now.Add(-time.Minute)),
	}
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	signed, err := tok.SignedString(testJWTSecret)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	resp := getMe(t, env, "Bearer "+signed)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "AUTH-011" {
		t.Fatalf("code = %q, want AUTH-011", er.Error.Code)
	}
}
