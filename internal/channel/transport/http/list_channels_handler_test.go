package httpchannel_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

func doGet(t *testing.T, env *testEnv, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, env.server.URL+path, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func TestListChannels_InvalidRoomUUID_Returns404(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doGet(t, env, "/rooms/bad-uuid/channels", tok)
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

func TestListChannels_NotMember_Returns403(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doGet(t, env, "/rooms/"+uuid.New().String()+"/channels", tok)
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

func TestListChannels_AsMember_Returns200(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	room := uuid.New()
	env.membership.withRole(room, uid, "member")
	env.channels.put(mustChannel(t, uuid.New(), room, "general", domain.ChannelKindText, env.clock.now))
	env.channels.put(mustChannel(t, uuid.New(), room, "voice", domain.ChannelKindVoice, env.clock.now))

	resp := doGet(t, env, "/rooms/"+room.String()+"/channels", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("items len = %d, want 2", len(body.Items))
	}
}

func TestListChannels_EmptyRoom_ReturnsEmptyArray(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	room := uuid.New()
	env.membership.withRole(room, uid, "member")

	resp := doGet(t, env, "/rooms/"+room.String()+"/channels", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items []any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 0 {
		t.Fatalf("items len = %d, want 0", len(body.Items))
	}
}
