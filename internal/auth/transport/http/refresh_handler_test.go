package httpauth_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

func loginAndGetTokens(t *testing.T, env *testEnv, email, password string) (string, string) {
	t.Helper()
	resp := postJSON(t, env, "/login",
		`{"email":"`+email+`","password":"`+password+`"}`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", resp.StatusCode)
	}
	var body struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.AccessToken, body.RefreshToken
}

func TestRefreshHandler_Success(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	registerUser(t, env, "user@example.com", "user_1", "correct-pass")
	_, rt := loginAndGetTokens(t, env, "user@example.com", "correct-pass")

	resp := postJSON(t, env, "/refresh",
		`{"refreshToken":"`+rt+`"}`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.AccessToken == "" || body.RefreshToken == "" {
		t.Fatalf("empty tokens: %+v", body)
	}
}

func TestRefreshHandler_MalformedBody(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	resp := postJSON(t, env, "/refresh", `{not`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "AUTH-012" {
		t.Fatalf("code = %q, want AUTH-012", er.Error.Code)
	}
}

func TestRefreshHandler_NotFound(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	rt := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	resp := postJSON(t, env, "/refresh", `{"refreshToken":"`+rt+`"}`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "AUTH-007" {
		t.Fatalf("code = %q, want AUTH-007", er.Error.Code)
	}
}

func TestRefreshHandler_Revoked(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	registerUser(t, env, "user@example.com", "user_1", "correct-pass")
	_, rt := loginAndGetTokens(t, env, "user@example.com", "correct-pass")

	rawDecoded, err := base64.RawURLEncoding.DecodeString(rt)
	if err != nil {
		t.Fatalf("decode rt: %v", err)
	}
	digest := sha256.Sum256(rawDecoded)
	hash, err := domain.NewTokenHash(digest[:])
	if err != nil {
		t.Fatalf("NewTokenHash: %v", err)
	}
	stored, err := env.refresh.FindByHash(context.Background(), hash)
	if err != nil {
		t.Fatalf("FindByHash: %v", err)
	}
	if revokeErr := stored.Revoke(env.clock.now); revokeErr != nil {
		t.Fatalf("Revoke: %v", revokeErr)
	}

	resp := postJSON(t, env, "/refresh", `{"refreshToken":"`+rt+`"}`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "AUTH-008" {
		t.Fatalf("code = %q, want AUTH-008", er.Error.Code)
	}
}

func TestRefreshHandler_Expired(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	registerUser(t, env, "user@example.com", "user_1", "correct-pass")
	_, rt := loginAndGetTokens(t, env, "user@example.com", "correct-pass")

	// Заменяем сохранённый токен на просроченный.
	rawDecoded, _ := base64.RawURLEncoding.DecodeString(rt)
	digest := sha256.Sum256(rawDecoded)
	hash, _ := domain.NewTokenHash(digest[:])

	expired, err := domain.NewRefreshToken(
		mustRefreshTokenIDForTest(t, uuid.New()),
		mustUserIDForTest(t, uuid.New()),
		hash,
		env.clock.now.Add(-time.Minute),
		env.clock.now.Add(-time.Hour),
	)
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	env.refresh.byHash[hash.Bytes()] = expired

	resp := postJSON(t, env, "/refresh", `{"refreshToken":"`+rt+`"}`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "AUTH-009" {
		t.Fatalf("code = %q, want AUTH-009", er.Error.Code)
	}
}

func mustUserIDForTest(t *testing.T, raw uuid.UUID) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("NewUserID: %v", err)
	}
	return id
}

func mustRefreshTokenIDForTest(t *testing.T, raw uuid.UUID) domain.RefreshTokenID {
	t.Helper()
	id, err := domain.NewRefreshTokenID(raw)
	if err != nil {
		t.Fatalf("NewRefreshTokenID: %v", err)
	}
	return id
}
