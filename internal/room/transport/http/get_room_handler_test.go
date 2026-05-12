package httproom_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
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

func TestGetRoom_InvalidUUID_Returns404(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doGet(t, env, "/rooms/not-a-uuid", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "ROOM-002" {
		t.Fatalf("code = %q, want ROOM-002", er.Error.Code)
	}
}

func TestGetRoom_AsMember_Returns200(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)

	roomUUID := uuid.New()
	env.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, uid, "general", env.clock.now)
	env.memberships.put(mustMembership(t, roomUUID, uid, domain.RoleMember, env.clock.now))

	resp := doGet(t, env, "/rooms/"+roomUUID.String(), tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestGetRoom_NotMember_Returns403(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())
	roomUUID := uuid.New()
	env.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, uuid.New(), "general", env.clock.now)

	resp := doGet(t, env, "/rooms/"+roomUUID.String(), tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "ROOM-003" {
		t.Fatalf("code = %q, want ROOM-003", er.Error.Code)
	}
}
