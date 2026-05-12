package httproom_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestListRooms_HasItems_Returns200(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)

	roomUUID := uuid.New()
	env.rooms.rooms[roomUUID] = mustRoom(t, roomUUID, uid, "g", env.clock.now)
	env.rooms.memberRoles[mkey{room: roomUUID, user: uid}] = domain.RoleOwner

	resp := doGet(t, env, "/rooms", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Role != "owner" {
		t.Fatalf("items = %+v, want 1 owner", body.Items)
	}
}

func TestListRooms_NoItems_ReturnsEmptyArray(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doGet(t, env, "/rooms", tok)
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
