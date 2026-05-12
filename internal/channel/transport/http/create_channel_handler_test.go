package httpchannel_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
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

func TestCreateChannel_BadBody_Returns400(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())
	room := uuid.New()

	req, err := http.NewRequest(http.MethodPost,
		env.server.URL+"/rooms/"+room.String()+"/channels",
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
	if er.Error.Code != "CHANNEL-005" {
		t.Fatalf("code = %q, want CHANNEL-005", er.Error.Code)
	}
}

func TestCreateChannel_InvalidRoomUUID_Returns404(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doPost(t, env, "/rooms/bad-uuid/channels", tok,
		map[string]string{"name": "general", "kind": "text"})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHANNEL-003" {
		t.Fatalf("code = %q, want CHANNEL-003", er.Error.Code)
	}
}

func TestCreateChannel_InvalidName_Returns400(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	room := uuid.New()
	env.membership.withRole(room, uid, "admin")

	resp := doPost(t, env, "/rooms/"+room.String()+"/channels", tok,
		map[string]string{"name": "", "kind": "text"})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHANNEL-001" {
		t.Fatalf("code = %q, want CHANNEL-001", er.Error.Code)
	}
}

func TestCreateChannel_InvalidKind_Returns400(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	room := uuid.New()
	env.membership.withRole(room, uid, "admin")

	resp := doPost(t, env, "/rooms/"+room.String()+"/channels", tok,
		map[string]string{"name": "general", "kind": "video"})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHANNEL-002" {
		t.Fatalf("code = %q, want CHANNEL-002", er.Error.Code)
	}
}

func TestCreateChannel_NotMember_Returns403_CHANNEL006(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doPost(t, env, "/rooms/"+uuid.New().String()+"/channels", tok,
		map[string]string{"name": "general", "kind": "text"})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHANNEL-006" {
		t.Fatalf("code = %q, want CHANNEL-006", er.Error.Code)
	}
}

func TestCreateChannel_AsMember_Returns403_CHANNEL007(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	room := uuid.New()
	env.membership.withRole(room, uid, "member")

	resp := doPost(t, env, "/rooms/"+room.String()+"/channels", tok,
		map[string]string{"name": "general", "kind": "text"})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHANNEL-007" {
		t.Fatalf("code = %q, want CHANNEL-007", er.Error.Code)
	}
}

func TestCreateChannel_AsAdmin_Returns201(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	room := uuid.New()
	env.membership.withRole(room, uid, "admin")

	resp := doPost(t, env, "/rooms/"+room.String()+"/channels", tok,
		map[string]string{"name": "general", "kind": "text"})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var body struct {
		ID     string `json:"id"`
		RoomID string `json:"roomId"`
		Name   string `json:"name"`
		Kind   string `json:"kind"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "general" || body.Kind != "text" {
		t.Fatalf("body mismatch: %+v", body)
	}
	if body.RoomID != room.String() {
		t.Fatalf("roomId = %q, want %q", body.RoomID, room.String())
	}
}

func TestCreateChannel_DuplicateName_Returns409(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	room := uuid.New()
	env.membership.withRole(room, uid, "admin")
	env.channels.put(mustChannel(t, uuid.New(), room, "general", domain.ChannelKindText, env.clock.now))

	resp := doPost(t, env, "/rooms/"+room.String()+"/channels", tok,
		map[string]string{"name": "general", "kind": "text"})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHANNEL-004" {
		t.Fatalf("code = %q, want CHANNEL-004", er.Error.Code)
	}
}
