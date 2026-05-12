package httproom_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestListMembers_InvalidUUID_Returns404(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doGet(t, env, "/rooms/bad-uuid/members", tok)
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

func TestListMembers_AsMember_Returns200(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	roomUUID := uuid.New()

	env.memberships.put(mustMembership(t, roomUUID, uid, domain.RoleMember, env.clock.now))
	env.memberships.put(mustMembership(t, roomUUID, uuid.New(), domain.RoleOwner, env.clock.now))

	resp := doGet(t, env, "/rooms/"+roomUUID.String()+"/members", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			UserID string `json:"userId"`
			Role   string `json:"role"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("items len = %d, want 2", len(body.Items))
	}
}
