package httproom_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
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

func TestDeleteRoom_NotFound_Returns404(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)

	// Membership есть, room — нет.
	roomUUID := uuid.New()
	env.memberships.put(mustMembership(t, roomUUID, uid, domain.RoleOwner, env.clock.now))

	resp := doDelete(t, env, "/rooms/"+roomUUID.String(), tok)
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

func TestDeleteRoom_AsOwner_Returns204(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)

	roomUUID := uuid.New()
	env.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, uid, "g", env.clock.now)
	env.memberships.put(mustMembership(t, roomUUID, uid, domain.RoleOwner, env.clock.now))

	resp := doDelete(t, env, "/rooms/"+roomUUID.String(), tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestDeleteRoom_AsAdmin_Returns403_ROOM005(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)

	roomUUID := uuid.New()
	env.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, uuid.New(), "g", env.clock.now)
	env.memberships.put(mustMembership(t, roomUUID, uid, domain.RoleAdmin, env.clock.now))

	resp := doDelete(t, env, "/rooms/"+roomUUID.String(), tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "ROOM-005" {
		t.Fatalf("code = %q, want ROOM-005", er.Error.Code)
	}
}

func TestDeleteRoom_NotMember_Returns403_ROOM003(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doDelete(t, env, "/rooms/"+uuid.New().String(), tok)
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
