package usecase_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

// ---------- Helpers ----------

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

// ---------- fakeInviteCodeGen ----------

type fakeInviteCodeGen struct {
	queue []string
	idx   int
	calls int
}

func (g *fakeInviteCodeGen) New() (domain.InviteCode, error) {
	g.calls++
	raw := "AAAAAAAA"
	if g.idx < len(g.queue) {
		raw = g.queue[g.idx]
		g.idx++
	}
	return domain.NewInviteCode(raw)
}

// ---------- fakeRoomRepo ----------

type fakeRoomRepo struct {
	rooms            map[uuid.UUID]*domain.Room
	ownerOf          map[uuid.UUID]uuid.UUID // roomID → ownerUserID
	memberRoles      map[mkey]domain.Role    // (roomID, userID) → role
	saved            []*domain.Room
	savedMemberships []*domain.Membership
	saveWithOwnerErr error
	findByIDErr      error
	listByMemberErr  error
	deleteErr        error
	deleted          []uuid.UUID
}

func newFakeRoomRepo() *fakeRoomRepo {
	return &fakeRoomRepo{
		rooms:       map[uuid.UUID]*domain.Room{},
		ownerOf:     map[uuid.UUID]uuid.UUID{},
		memberRoles: map[mkey]domain.Role{},
	}
}

func (r *fakeRoomRepo) SaveWithOwner(_ context.Context, room *domain.Room, owner *domain.Membership) error {
	if r.saveWithOwnerErr != nil {
		return r.saveWithOwnerErr
	}
	r.rooms[room.ID().UUID()] = room
	r.ownerOf[room.ID().UUID()] = owner.UserID().UUID()
	r.memberRoles[mkey{room: room.ID().UUID(), user: owner.UserID().UUID()}] = owner.Role()
	r.saved = append(r.saved, room)
	r.savedMemberships = append(r.savedMemberships, owner)
	return nil
}

func (r *fakeRoomRepo) FindByID(_ context.Context, id domain.RoomID) (*domain.Room, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	room, ok := r.rooms[id.UUID()]
	if !ok {
		return nil, domain.ErrRoomNotFound
	}
	return room, nil
}

func (r *fakeRoomRepo) ListByMember(_ context.Context, userID domain.UserID) ([]usecase.RoomWithRole, error) {
	if r.listByMemberErr != nil {
		return nil, r.listByMemberErr
	}
	var out []usecase.RoomWithRole
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
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if _, ok := r.rooms[id.UUID()]; !ok {
		return domain.ErrRoomNotFound
	}
	delete(r.rooms, id.UUID())
	delete(r.ownerOf, id.UUID())
	for k := range r.memberRoles {
		if k.room == id.UUID() {
			delete(r.memberRoles, k)
		}
	}
	r.deleted = append(r.deleted, id.UUID())
	return nil
}

// ---------- fakeMembershipRepo ----------

type mkey struct {
	room uuid.UUID
	user uuid.UUID
}

type fakeMembershipRepo struct {
	data          map[mkey]*domain.Membership
	added         []*domain.Membership
	findErr       error
	listByRoomErr error
	addErr        error
}

func newFakeMembershipRepo() *fakeMembershipRepo {
	return &fakeMembershipRepo{data: map[mkey]*domain.Membership{}}
}

func (r *fakeMembershipRepo) FindByPair(_ context.Context, roomID domain.RoomID, userID domain.UserID) (*domain.Membership, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	m, ok := r.data[mkey{room: roomID.UUID(), user: userID.UUID()}]
	if !ok {
		return nil, domain.ErrNotMember
	}
	return m, nil
}

func (r *fakeMembershipRepo) ListByRoom(_ context.Context, roomID domain.RoomID) ([]*domain.Membership, error) {
	if r.listByRoomErr != nil {
		return nil, r.listByRoomErr
	}
	var out []*domain.Membership
	for k, m := range r.data {
		if k.room == roomID.UUID() {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *fakeMembershipRepo) Add(_ context.Context, m *domain.Membership) error {
	if r.addErr != nil {
		return r.addErr
	}
	key := mkey{room: m.RoomID().UUID(), user: m.UserID().UUID()}
	if _, exists := r.data[key]; exists {
		return domain.ErrAlreadyMember
	}
	r.data[key] = m
	r.added = append(r.added, m)
	return nil
}

func (r *fakeMembershipRepo) put(m *domain.Membership) {
	r.data[mkey{room: m.RoomID().UUID(), user: m.UserID().UUID()}] = m
}

// ---------- fakeInviteRepo ----------

type fakeInviteRepo struct {
	byID                map[uuid.UUID]*domain.Invite
	nextRegenCollisions int
	regenCalls          int
	findErr             error
	regenErr            error
}

func newFakeInviteRepo() *fakeInviteRepo {
	return &fakeInviteRepo{byID: map[uuid.UUID]*domain.Invite{}}
}

func (r *fakeInviteRepo) RegenerateActive(_ context.Context, inv *domain.Invite) error {
	r.regenCalls++
	if r.regenErr != nil {
		return r.regenErr
	}
	if r.nextRegenCollisions > 0 {
		r.nextRegenCollisions--
		return usecase.ErrInviteCodeCollision
	}
	// Отозвать все активные invites текущей комнаты.
	for id, existing := range r.byID {
		if existing.RoomID() == inv.RoomID() && existing.IsActive() {
			_ = existing.Revoke(time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC))
			r.byID[id] = existing
		}
	}
	r.byID[inv.ID().UUID()] = inv
	return nil
}

func (r *fakeInviteRepo) FindActiveByCode(_ context.Context, code domain.InviteCode) (*domain.Invite, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	for _, inv := range r.byID {
		if inv.IsActive() && inv.Code() == code {
			return inv, nil
		}
	}
	return nil, domain.ErrInviteNotFound
}

func (r *fakeInviteRepo) put(inv *domain.Invite) {
	r.byID[inv.ID().UUID()] = inv
}

// ---------- fakeRoomEventsPublisher ----------

type memberJoinedCall struct {
	RoomID   domain.RoomID
	UserID   domain.UserID
	JoinedAt time.Time
}

type fakeRoomEventsPublisher struct {
	mu           sync.Mutex
	memberJoined []memberJoinedCall
	panicNext    bool
}

func newFakeRoomEventsPublisher() *fakeRoomEventsPublisher {
	return &fakeRoomEventsPublisher{}
}

func (p *fakeRoomEventsPublisher) PublishMemberJoined(roomID domain.RoomID, userID domain.UserID, joinedAt time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.panicNext {
		p.panicNext = false
		panic("simulated publisher panic")
	}
	p.memberJoined = append(p.memberJoined, memberJoinedCall{RoomID: roomID, UserID: userID, JoinedAt: joinedAt})
}

func (p *fakeRoomEventsPublisher) calls() []memberJoinedCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]memberJoinedCall, len(p.memberJoined))
	copy(out, p.memberJoined)
	return out
}
