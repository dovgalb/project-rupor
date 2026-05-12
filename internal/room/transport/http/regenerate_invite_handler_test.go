package httproom_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestRegenerateInvite_InvalidUUID_Returns404(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doPost(t, env, "/rooms/bad-uuid/invite", tok, nil)
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

func TestRegenerateInvite_AsMember_Returns403(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	roomUUID := uuid.New()
	env.memberships.put(mustMembership(t, roomUUID, uid, domain.RoleMember, env.clock.now))

	resp := doPost(t, env, "/rooms/"+roomUUID.String()+"/invite", tok, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "ROOM-004" {
		t.Fatalf("code = %q, want ROOM-004", er.Error.Code)
	}
}

func TestRegenerateInvite_AsAdmin_Returns200(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	roomUUID := uuid.New()
	env.memberships.put(mustMembership(t, roomUUID, uid, domain.RoleAdmin, env.clock.now))

	resp := doPost(t, env, "/rooms/"+roomUUID.String()+"/invite", tok, nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Code      string `json:"code"`
		CreatedBy string `json:"createdBy"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Code) != 8 {
		t.Fatalf("code len = %d, want 8", len(body.Code))
	}
}
