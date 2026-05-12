package httpchannel_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

func doDelete(t *testing.T, env *testEnv, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, env.server.URL+path, nil)
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

func TestDeleteChannel_InvalidUUID_Returns404(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doDelete(t, env, "/rooms/"+uuid.New().String()+"/channels/bad-uuid", tok)
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

func TestDeleteChannel_NotFound_Returns404(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	room := uuid.New()
	env.membership.withRole(room, uid, "admin")

	resp := doDelete(t, env,
		"/rooms/"+room.String()+"/channels/"+uuid.New().String(), tok)
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

func TestDeleteChannel_NotMember_Returns403(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doDelete(t, env,
		"/rooms/"+uuid.New().String()+"/channels/"+uuid.New().String(), tok)
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

func TestDeleteChannel_AsMember_Returns403_CHANNEL007(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	room := uuid.New()
	env.membership.withRole(room, uid, "member")

	resp := doDelete(t, env,
		"/rooms/"+room.String()+"/channels/"+uuid.New().String(), tok)
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

func TestDeleteChannel_AsAdmin_Returns204(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	room := uuid.New()
	env.membership.withRole(room, uid, "admin")
	chUUID := uuid.New()
	env.channels.put(mustChannel(t, chUUID, room, "general", domain.ChannelKindText, env.clock.now))

	resp := doDelete(t, env,
		"/rooms/"+room.String()+"/channels/"+chUUID.String(), tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}
