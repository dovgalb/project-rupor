package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
	"github.com/dovgalb/project-rupor/internal/channel/usecase"
)

// ---------- Helpers ----------

func mustChannelID(t *testing.T, raw uuid.UUID) domain.ChannelID {
	t.Helper()
	id, err := domain.NewChannelID(raw)
	if err != nil {
		t.Fatalf("mustChannelID: %v", err)
	}
	return id
}

func mustRoomID(t *testing.T, raw uuid.UUID) domain.RoomID {
	t.Helper()
	id, err := domain.NewRoomID(raw)
	if err != nil {
		t.Fatalf("mustRoomID: %v", err)
	}
	return id
}

func mustChannelName(t *testing.T, raw string) domain.ChannelName {
	t.Helper()
	n, err := domain.NewChannelName(raw)
	if err != nil {
		t.Fatalf("mustChannelName: %v", err)
	}
	return n
}

func mustChannel(t *testing.T, id, roomID uuid.UUID, name string, kind domain.ChannelKind, createdAt time.Time) *domain.Channel {
	t.Helper()
	ch, err := domain.NewChannel(
		mustChannelID(t, id),
		mustRoomID(t, roomID),
		mustChannelName(t, name),
		kind,
		createdAt,
	)
	if err != nil {
		t.Fatalf("mustChannel: %v", err)
	}
	return ch
}

// ---------- fixedClock / fixedUUID ----------

type fixedClock struct{ now time.Time }

func (c *fixedClock) Now() time.Time { return c.now }

type fixedUUID struct {
	next []uuid.UUID
	idx  int
}

func (g *fixedUUID) New() uuid.UUID {
	id := g.next[g.idx]
	g.idx++
	return id
}

// ---------- fakeChannelRepo ----------

type chKey struct {
	room uuid.UUID
	name string
}

type fakeChannelRepo struct {
	byID    map[uuid.UUID]*domain.Channel
	byName  map[chKey]uuid.UUID
	saved   []*domain.Channel
	deleted []uuid.UUID
	saveErr error
	listErr error
	delErr  error
}

func newFakeChannelRepo() *fakeChannelRepo {
	return &fakeChannelRepo{
		byID:   map[uuid.UUID]*domain.Channel{},
		byName: map[chKey]uuid.UUID{},
	}
}

func (r *fakeChannelRepo) Save(_ context.Context, ch *domain.Channel) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	key := chKey{room: ch.RoomID().UUID(), name: ch.Name().String()}
	if _, exists := r.byName[key]; exists {
		return domain.ErrChannelNameAlreadyTaken
	}
	r.byID[ch.ID().UUID()] = ch
	r.byName[key] = ch.ID().UUID()
	r.saved = append(r.saved, ch)
	return nil
}

func (r *fakeChannelRepo) ListByRoom(_ context.Context, roomID domain.RoomID) ([]*domain.Channel, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []*domain.Channel
	for _, ch := range r.byID {
		if ch.RoomID() == roomID {
			out = append(out, ch)
		}
	}
	return out, nil
}

func (r *fakeChannelRepo) DeleteInRoom(_ context.Context, channelID domain.ChannelID, roomID domain.RoomID) error {
	if r.delErr != nil {
		return r.delErr
	}
	ch, ok := r.byID[channelID.UUID()]
	if !ok || ch.RoomID() != roomID {
		return domain.ErrChannelNotFound
	}
	delete(r.byID, channelID.UUID())
	delete(r.byName, chKey{room: ch.RoomID().UUID(), name: ch.Name().String()})
	r.deleted = append(r.deleted, channelID.UUID())
	return nil
}

func (r *fakeChannelRepo) put(ch *domain.Channel) {
	r.byID[ch.ID().UUID()] = ch
	r.byName[chKey{room: ch.RoomID().UUID(), name: ch.Name().String()}] = ch.ID().UUID()
}

// ---------- fakeMembershipQuery ----------

// testRole — внутренний enum для фейка, чтобы не зависеть от room.domain.Role.
type testRole int

const (
	roleNone testRole = iota
	roleMember
	roleAdmin
	roleOwner
)

type mqKey struct {
	room uuid.UUID
	user uuid.UUID
}

type fakeMembershipQuery struct {
	roles      map[mqKey]testRole
	requireErr error // принудительная техническая ошибка (для теста wrap)
	calls      int
}

func newFakeMembershipQuery() *fakeMembershipQuery {
	return &fakeMembershipQuery{roles: map[mqKey]testRole{}}
}

func (m *fakeMembershipQuery) withRole(roomID, userID uuid.UUID, role string) *fakeMembershipQuery {
	var r testRole
	switch role {
	case "owner":
		r = roleOwner
	case "admin":
		r = roleAdmin
	case "member":
		r = roleMember
	default:
		r = roleNone
	}
	m.roles[mqKey{room: roomID, user: userID}] = r
	return m
}

func (m *fakeMembershipQuery) Require(_ context.Context, roomID domain.RoomID, userID domain.UserID, req usecase.RoleRequirement) error {
	m.calls++
	if m.requireErr != nil {
		return m.requireErr
	}
	role, ok := m.roles[mqKey{room: roomID.UUID(), user: userID.UUID()}]
	if !ok || role == roleNone {
		return domain.ErrChannelAccessDenied
	}
	if !roleSatisfies(role, req) {
		return domain.ErrChannelInsufficientRole
	}
	return nil
}

func roleSatisfies(role testRole, req usecase.RoleRequirement) bool {
	switch req {
	case usecase.RoleAnyMember:
		return role == roleMember || role == roleAdmin || role == roleOwner
	case usecase.RoleAdminOrOwner:
		return role == roleAdmin || role == roleOwner
	case usecase.RoleOwnerOnly:
		return role == roleOwner
	default:
		return false
	}
}
