package httpchannel_test

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	jwtrepo "github.com/dovgalb/project-rupor/internal/auth/repository/jwt"
	"github.com/dovgalb/project-rupor/internal/channel/domain"
	httpchannel "github.com/dovgalb/project-rupor/internal/channel/transport/http"
	"github.com/dovgalb/project-rupor/internal/channel/usecase"
)

const testAccessTTL = 15 * time.Minute

var testJWTSecret = []byte("test-secret-1234567890")

// ---------- helpers ----------

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

// ---------- realClock / seqUUID ----------

type realClock struct{ now time.Time }

func (c *realClock) Now() time.Time { return c.now }

type seqUUID struct {
	mu  sync.Mutex
	nxt []uuid.UUID
	idx int
}

func (g *seqUUID) New() uuid.UUID {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.idx < len(g.nxt) {
		id := g.nxt[g.idx]
		g.idx++
		return id
	}
	return uuid.New()
}

// ---------- fakes (копии из channel/usecase/fakes_test.go с sync.Mutex для httptest) ----------

type chKey struct {
	room uuid.UUID
	name string
}

type fakeChannelRepo struct {
	mu     sync.Mutex
	byID   map[uuid.UUID]*domain.Channel
	byName map[chKey]uuid.UUID
}

func newFakeChannelRepo() *fakeChannelRepo {
	return &fakeChannelRepo{
		byID:   map[uuid.UUID]*domain.Channel{},
		byName: map[chKey]uuid.UUID{},
	}
}

func (r *fakeChannelRepo) Save(_ context.Context, ch *domain.Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := chKey{room: ch.RoomID().UUID(), name: ch.Name().String()}
	if _, exists := r.byName[key]; exists {
		return domain.ErrChannelNameAlreadyTaken
	}
	r.byID[ch.ID().UUID()] = ch
	r.byName[key] = ch.ID().UUID()
	return nil
}

func (r *fakeChannelRepo) ListByRoom(_ context.Context, roomID domain.RoomID) ([]*domain.Channel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*domain.Channel, 0)
	for _, ch := range r.byID {
		if ch.RoomID() == roomID {
			out = append(out, ch)
		}
	}
	return out, nil
}

func (r *fakeChannelRepo) DeleteInRoom(_ context.Context, channelID domain.ChannelID, roomID domain.RoomID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch, ok := r.byID[channelID.UUID()]
	if !ok || ch.RoomID() != roomID {
		return domain.ErrChannelNotFound
	}
	delete(r.byID, channelID.UUID())
	delete(r.byName, chKey{room: ch.RoomID().UUID(), name: ch.Name().String()})
	return nil
}

func (r *fakeChannelRepo) put(ch *domain.Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[ch.ID().UUID()] = ch
	r.byName[chKey{room: ch.RoomID().UUID(), name: ch.Name().String()}] = ch.ID().UUID()
}

// testRole — внутренний enum фейка (без зависимости от room.domain.Role).
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
	mu    sync.Mutex
	roles map[mqKey]testRole
}

func newFakeMembershipQuery() *fakeMembershipQuery {
	return &fakeMembershipQuery{roles: map[mqKey]testRole{}}
}

func (m *fakeMembershipQuery) withRole(roomID, userID uuid.UUID, role string) *fakeMembershipQuery {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.Lock()
	defer m.mu.Unlock()
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

// ---------- testEnv ----------

type testEnv struct {
	server     *httptest.Server
	channels   *fakeChannelRepo
	membership *fakeMembershipQuery
	clock      *realClock
	uuids      *seqUUID
	issuer     *jwtrepo.TokenIssuer
}

func setupServer(t *testing.T) *testEnv {
	t.Helper()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	clock := &realClock{now: now}
	uuids := &seqUUID{}
	issuer := jwtrepo.NewTokenIssuer(testJWTSecret, testAccessTTL)

	channels := newFakeChannelRepo()
	membership := newFakeMembershipQuery()

	createChannel := usecase.NewCreateChannel(channels, membership, clock, uuids)
	listChannels := usecase.NewListChannels(channels, membership)
	deleteChannel := usecase.NewDeleteChannel(channels, membership)

	r := chi.NewRouter()
	httpchannel.RegisterRoutes(r, httpchannel.Deps{
		CreateChannel: createChannel,
		ListChannels:  listChannels,
		DeleteChannel: deleteChannel,
		TokenIssuer:   issuer,
		Clock:         clock,
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &testEnv{
		server: srv, channels: channels, membership: membership,
		clock: clock, uuids: uuids, issuer: issuer,
	}
}

func issueTestToken(t *testing.T, env *testEnv, userID uuid.UUID) string {
	t.Helper()
	uid, err := authdom.NewUserID(userID)
	if err != nil {
		t.Fatalf("auth NewUserID: %v", err)
	}
	tok, _, err := env.issuer.IssueAccess(uid, env.clock.now)
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	return tok
}

// errResp для парсинга {"error":{"code","message"}}.
type errResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
