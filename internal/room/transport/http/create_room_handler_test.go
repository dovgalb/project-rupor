package httproom_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func doPost(t *testing.T, env *testEnv, path, token string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(http.MethodPost, env.server.URL+path, &buf)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func TestCreateRoom_Valid_Returns201(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)

	resp := doPost(t, env, "/rooms", tok, map[string]string{"name": "general"})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var body struct {
		ID        string    `json:"id"`
		OwnerID   string    `json:"ownerId"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "general" {
		t.Fatalf("name = %q, want general", body.Name)
	}
	if body.OwnerID != uid.String() {
		t.Fatalf("ownerId = %q, want %q", body.OwnerID, uid.String())
	}
}

func TestCreateRoom_BadBody_Returns400(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	req, err := http.NewRequest(http.MethodPost, env.server.URL+"/rooms",
		strings.NewReader("{not json}"))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "ROOM-009" {
		t.Fatalf("code = %q, want ROOM-009", er.Error.Code)
	}
}

func TestCreateRoom_NoToken_Returns401(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	resp := doPost(t, env, "/rooms", "", map[string]string{"name": "general"})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestCreateRoom_InvalidName_Returns400(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doPost(t, env, "/rooms", tok, map[string]string{"name": ""})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "ROOM-001" {
		t.Fatalf("code = %q, want ROOM-001", er.Error.Code)
	}
}
