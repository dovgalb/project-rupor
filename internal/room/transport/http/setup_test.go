package httproom_test

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
	authuc "github.com/dovgalb/project-rupor/internal/auth/usecase"
	"github.com/dovgalb/project-rupor/internal/room/domain"
	httproom "github.com/dovgalb/project-rupor/internal/room/transport/http"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

const testAccessTTL = 15 * time.Minute

var testJWTSecret = []byte("test-secret-1234567890")

// ---------- helpers ----------

func mustRoomID(t *testing.T, raw uuid.UUID) domain.RoomID {
	t.Helper()
	id, err := domain.NewRoomID(raw)
	if err != nil {
		t.Fatalf("mustRoomID: %v", err)
	}
	return id
}

func mustUserID(t *testing.T, raw uuid.UUID) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("mustUserID: %v", err)
	}
	return id
}

func mustInviteID(t *testing.T, raw uuid.UUID) domain.InviteID {
	t.Helper()
	id, err := domain.NewInviteID(raw)
	if err != nil {
		t.Fatalf("mustInviteID: %v", err)
	}
	return id
}

func mustRoomName(t *testing.T, raw string) domain.RoomName {
	t.Helper()
	n, err := domain.NewRoomName(raw)
	if err != nil {
		t.Fatalf("mustRoomName: %v", err)
	}
	return n
}

func mustInviteCode(t *testing.T, raw string) domain.InviteCode {
	t.Helper()
	c, err := domain.NewInviteCode(raw)
	if err != nil {
		t.Fatalf("mustInviteCode: %v", err)
	}
	return c
}

func mustRoom(t *testing.T, id, owner uuid.UUID, name string, createdAt time.Time) *domain.Room {
	t.Helper()
	r, err := domain.NewRoom(mustRoomID(t, id), mustUserID(t, owner), mustRoomName(t, name), createdAt)
	if err != nil {
		t.Fatalf("mustRoom: %v", err)
	}
	return r
}

func mustMembership(t *testing.T, roomID, userID uuid.UUID, role domain.Role, joinedAt time.Time) *domain.Membership {
	t.Helper()
	m, err := domain.NewMembership(mustRoomID(t, roomID), mustUserID(t, userID), role, joinedAt)
	if err != nil {
		t.Fatalf("mustMembership: %v", err)
	}
	return m
}

func mustInvite(t *testing.T, id, roomID uuid.UUID, code string, createdBy uuid.UUID, createdAt time.Time) *domain.Invite {
	t.Helper()
	inv, err := domain.NewInvite(
		mustInviteID(t, id),
		mustRoomID(t, roomID),
		mustInviteCode(t, code),
		mustUserID(t, createdBy),
		createdAt,
	)
	if err != nil {
		t.Fatalf("mustInvite: %v", err)
	}
	return inv
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

// ---------- fakeInviteCodeGen ----------

type fakeInviteCodeGen struct {
	mu    sync.Mutex
	queue []string
	idx   int
}

func (g *fakeInviteCodeGen) New() (domain.InviteCode, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	raw := "ABCDEFGH"
	if g.idx < len(g.queue) {
		raw = g.queue[g.idx]
		g.idx++
	}
	return domain.NewInviteCode(raw)
}

// ---------- fakes (копии из usecase, упрощённые до нужного для HTTP-тестов) ----------

type mkey struct {
	room uuid.UUID
	user uuid.UUID
}

type fakeRoomRepo struct {
	mu               sync.Mutex
	rooms            map[uuid.UUID]*domain.Room
	memberRoles      map[mkey]domain.Role
	saveWithOwnerErr error
	deleteErr        error
}

func newFakeRoomRepo() *fakeRoomRepo {
	return &fakeRoomRepo{
		rooms:       map[uuid.UUID]*domain.Room{},
		memberRoles: map[mkey]domain.Role{},
	}
}

func (r *fakeRoomRepo) SaveWithOwner(_ context.Context, room *domain.Room, owner *domain.Membership) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveWithOwnerErr != nil {
		return r.saveWithOwnerErr
	}
	r.rooms[room.ID().UUID()] = room
	r.memberRoles[mkey{room: room.ID().UUID(), user: owner.UserID().UUID()}] = owner.Role()
	return nil
}

func (r *fakeRoomRepo) FindByID(_ context.Context, id domain.RoomID) (*domain.Room, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	room, ok := r.rooms[id.UUID()]
	if !ok {
		return nil, domain.ErrRoomNotFound
	}
	return room, nil
}

func (r *fakeRoomRepo) ListByMember(_ context.Context, userID domain.UserID) ([]usecase.RoomWithRole, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]usecase.RoomWithRole, 0)
	for k, role := range r.memberRoles {
		if k.user != userID.UUID() {
			continue
		}
		room, ok := r.rooms[k.room]
		if !ok {
			continue
		}
		out = append(out, usecase.RoomWithRole{Room: room, Role: role})
	}
	return out, nil
}

func (r *fakeRoomRepo) Delete(_ context.Context, id domain.RoomID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if _, ok := r.rooms[id.UUID()]; !ok {
		return domain.ErrRoomNotFound
	}
	delete(r.rooms, id.UUID())
	for k := range r.memberRoles {
		if k.room == id.UUID() {
			delete(r.memberRoles, k)
		}
	}
	return nil
}

type fakeMembershipRepo struct {
	mu   sync.Mutex
	data map[mkey]*domain.Membership
}

func newFakeMembershipRepo() *fakeMembershipRepo {
	return &fakeMembershipRepo{data: map[mkey]*domain.Membership{}}
}

func (r *fakeMembershipRepo) FindByPair(_ context.Context, roomID domain.RoomID, userID domain.UserID) (*domain.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.data[mkey{room: roomID.UUID(), user: userID.UUID()}]
	if !ok {
		return nil, domain.ErrNotMember
	}
	return m, nil
}

func (r *fakeMembershipRepo) ListByRoom(_ context.Context, roomID domain.RoomID) ([]*domain.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*domain.Membership, 0)
	for k, m := range r.data {
		if k.room == roomID.UUID() {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *fakeMembershipRepo) Add(_ context.Context, m *domain.Membership) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := mkey{room: m.RoomID().UUID(), user: m.UserID().UUID()}
	if _, exists := r.data[key]; exists {
		return domain.ErrAlreadyMember
	}
	r.data[key] = m
	return nil
}

func (r *fakeMembershipRepo) put(m *domain.Membership) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[mkey{room: m.RoomID().UUID(), user: m.UserID().UUID()}] = m
}

type fakeInviteRepo struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*domain.Invite
}

func newFakeInviteRepo() *fakeInviteRepo {
	return &fakeInviteRepo{byID: map[uuid.UUID]*domain.Invite{}}
}

func (r *fakeInviteRepo) RegenerateActive(_ context.Context, inv *domain.Invite) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, existing := range r.byID {
		if existing.RoomID() == inv.RoomID() && existing.IsActive() {
			_ = existing.Revoke(inv.CreatedAt())
			r.byID[id] = existing
		}
	}
	r.byID[inv.ID().UUID()] = inv
	return nil
}

func (r *fakeInviteRepo) FindActiveByCode(_ context.Context, code domain.InviteCode) (*domain.Invite, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inv := range r.byID {
		if inv.IsActive() && inv.Code() == code {
			return inv, nil
		}
	}
	return nil, domain.ErrInviteNotFound
}

func (r *fakeInviteRepo) put(inv *domain.Invite) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[inv.ID().UUID()] = inv
}

// ---------- testEnv ----------

type testEnv struct {
	server      *httptest.Server
	rooms       *fakeRoomRepo
	memberships *fakeMembershipRepo
	invites     *fakeInviteRepo
	codes       *fakeInviteCodeGen
	clock       *realClock
	uuids       *seqUUID
	issuer      *jwtrepo.TokenIssuer
}

func setupServer(t *testing.T) *testEnv {
	t.Helper()

	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	clock := &realClock{now: now}
	uuids := &seqUUID{}
	issuer := jwtrepo.NewTokenIssuer(testJWTSecret, testAccessTTL)

	rooms := newFakeRoomRepo()
	memberships := newFakeMembershipRepo()
	invites := newFakeInviteRepo()
	codes := &fakeInviteCodeGen{}

	createRoom := usecase.NewCreateRoom(rooms, clock, uuids)
	getRoom := usecase.NewGetRoom(rooms, memberships)
	listUserRooms := usecase.NewListUserRooms(rooms)
	deleteRoom := usecase.NewDeleteRoom(rooms, memberships)
	listMembers := usecase.NewListMembers(memberships)
	regenInvite := usecase.NewRegenerateInvite(invites, memberships, codes, clock, uuids)
	joinByCode := usecase.NewJoinByCode(invites, memberships, rooms, clock, noopRoomEventsPublisher{})

	r := chi.NewRouter()
	httproom.RegisterRoutes(r, httproom.Deps{
		CreateRoom:       createRoom,
		GetRoom:          getRoom,
		ListUserRooms:    listUserRooms,
		DeleteRoom:       deleteRoom,
		ListMembers:      listMembers,
		RegenerateInvite: regenInvite,
		JoinByCode:       joinByCode,
		TokenIssuer:      issuer,
		Clock:            clock,
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &testEnv{
		server: srv, rooms: rooms, memberships: memberships, invites: invites,
		codes: codes, clock: clock, uuids: uuids, issuer: issuer,
	}
}

// issueTestToken генерирует Bearer-токен от имени userID.
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

// ensure authuc.Clock interface is satisfied (заглушка для линтера, не используется).
var _ authuc.Clock = (*realClock)(nil)

// errResp используется тестами для парсинга {"error":{"code","message"}}.
type errResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// noopRoomEventsPublisher — заглушка publisher'а для HTTP-тестов room.
// Тесты room-handler'ов не проверяют публикацию member.joined — для этого
// есть юнит-тесты JoinByCode.
type noopRoomEventsPublisher struct{}

func (noopRoomEventsPublisher) PublishMemberJoined(_ domain.RoomID, _ domain.UserID, _ time.Time) {
}
