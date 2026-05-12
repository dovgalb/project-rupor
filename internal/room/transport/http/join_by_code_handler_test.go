package httproom_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestJoinByCode_InvalidFormat_Returns400(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doPost(t, env, "/rooms/join/ABC", tok, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "ROOM-008" {
		t.Fatalf("code = %q, want ROOM-008", er.Error.Code)
	}
}

func TestJoinByCode_NoActiveInvite_Returns404(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doPost(t, env, "/rooms/join/ABCDEFGH", tok, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "ROOM-007" {
		t.Fatalf("code = %q, want ROOM-007", er.Error.Code)
	}
}

func TestJoinByCode_AlreadyMember_Returns409(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	owner := uuid.New()
	roomUUID := uuid.New()

	env.invites.put(mustInvite(t, uuid.New(), roomUUID, "ABCDEFGH", owner, env.clock.now))
	env.memberships.put(mustMembership(t, roomUUID, uid, domain.RoleMember, env.clock.now))

	resp := doPost(t, env, "/rooms/join/ABCDEFGH", tok, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "ROOM-006" {
		t.Fatalf("code = %q, want ROOM-006", er.Error.Code)
	}
}

func TestJoinByCode_NewMember_Returns200(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	owner := uuid.New()
	roomUUID := uuid.New()

	env.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, owner, "g", env.clock.now)
	env.invites.put(mustInvite(t, uuid.New(), roomUUID, "ABCDEFGH", owner, env.clock.now))

	resp := doPost(t, env, "/rooms/join/ABCDEFGH", tok, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ID != roomUUID.String() {
		t.Fatalf("ID = %q, want %q", body.ID, roomUUID.String())
	}
}
