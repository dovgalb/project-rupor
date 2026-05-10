package httpauth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type errResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func postJSON(t *testing.T, env *testEnv, path, body string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, env.server.URL+"/auth"+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func TestRegisterHandler_Success(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	resp := postJSON(t, env, "/register",
		`{"email":"user@example.com","username":"user_1","password":"12345678"}`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var body struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Username  string `json:"username"`
		CreatedAt string `json:"createdAt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, err := uuid.Parse(body.ID); err != nil {
		t.Fatalf("ID is not a UUID: %q", body.ID)
	}
	if body.Email != "user@example.com" || body.Username != "user_1" {
		t.Fatalf("body mismatch: %+v", body)
	}
	if body.CreatedAt == "" {
		t.Fatalf("CreatedAt is empty")
	}
}

func TestRegisterHandler_MalformedBody(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	resp := postJSON(t, env, "/register", `{not-json`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if er.Error.Code != "AUTH-012" {
		t.Fatalf("code = %q, want AUTH-012", er.Error.Code)
	}
}

func TestRegisterHandler_InvalidEmail(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	resp := postJSON(t, env, "/register",
		`{"email":"x","username":"user_1","password":"12345678"}`, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "AUTH-001" {
		t.Fatalf("code = %q, want AUTH-001", er.Error.Code)
	}
}

func TestRegisterHandler_EmailTaken(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	body := `{"email":"user@example.com","username":"user_1","password":"12345678"}`

	resp1 := postJSON(t, env, "/register", body, nil)
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first register status = %d", resp1.StatusCode)
	}

	resp2 := postJSON(t, env, "/register",
		`{"email":"user@example.com","username":"user_2","password":"12345678"}`, nil)
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp2.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp2.Body).Decode(&er)
	if er.Error.Code != "AUTH-004" {
		t.Fatalf("code = %q, want AUTH-004", er.Error.Code)
	}
}

func TestRegisterHandler_UnsupportedMethod(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	req, err := http.NewRequest(http.MethodGet, env.server.URL+"/auth/register", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}
