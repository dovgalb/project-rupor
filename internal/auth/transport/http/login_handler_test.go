package httpauth_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func registerUser(t *testing.T, env *testEnv, email, username, password string) {
	t.Helper()
	body := `{"email":"` + email + `","username":"` + username + `","password":"` + password + `"}`
	resp := postJSON(t, env, "/register", body, nil)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", resp.StatusCode)
	}
}

func TestLoginHandler_Success(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	registerUser(t, env, "user@example.com", "user_1", "correct-pass")

	resp := postJSON(t, env, "/login",
		`{"email":"user@example.com","password":"correct-pass"}`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		AccessToken      string `json:"accessToken"`
		RefreshToken     string `json:"refreshToken"`
		AccessExpiresAt  string `json:"accessExpiresAt"`
		RefreshExpiresAt string `json:"refreshExpiresAt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.AccessToken == "" || body.RefreshToken == "" {
		t.Fatalf("empty tokens: %+v", body)
	}
	if body.AccessExpiresAt == "" || body.RefreshExpiresAt == "" {
		t.Fatalf("empty exp: %+v", body)
	}
}

func TestLoginHandler_MalformedBody(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	resp := postJSON(t, env, "/login", `{`, nil)
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

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{"bad email format", `{"email":"x","password":"12345678"}`},
		{"user not found", `{"email":"missing@example.com","password":"12345678"}`},
		{"wrong password", `{"email":"user@example.com","password":"wrong-pass"}`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := setupServer(t)
			registerUser(t, env, "user@example.com", "user_1", "correct-pass")

			resp := postJSON(t, env, "/login", tc.body, nil)
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", resp.StatusCode)
			}
			var er errResp
			_ = json.NewDecoder(resp.Body).Decode(&er)
			if er.Error.Code != "AUTH-006" {
				t.Fatalf("code = %q, want AUTH-006", er.Error.Code)
			}
		})
	}
}

func TestLoginHandler_InternalError(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	registerUser(t, env, "user@example.com", "user_1", "correct-pass")
	env.rand.err = errors.New("rand boom")

	resp := postJSON(t, env, "/login",
		`{"email":"user@example.com","password":"correct-pass"}`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "INTERNAL" {
		t.Fatalf("code = %q, want INTERNAL", er.Error.Code)
	}
}
