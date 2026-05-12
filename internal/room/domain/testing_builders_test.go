package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

// Фиксированное время — общий якорь детерминизма для всех билдеров фазы.
var defaultBuilderTime = time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)

func mustRoomID(t *testing.T, raw uuid.UUID) domain.RoomID {
	t.Helper()
	id, err := domain.NewRoomID(raw)
	if err != nil {
		t.Fatalf("mustRoomID(%s): %v", raw, err)
	}
	return id
}

func mustUserID(t *testing.T, raw uuid.UUID) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("mustUserID(%s): %v", raw, err)
	}
	return id
}

func mustInviteID(t *testing.T, raw uuid.UUID) domain.InviteID {
	t.Helper()
	id, err := domain.NewInviteID(raw)
	if err != nil {
		t.Fatalf("mustInviteID(%s): %v", raw, err)
	}
	return id
}

func mustRoomName(t *testing.T, raw string) domain.RoomName {
	t.Helper()
	n, err := domain.NewRoomName(raw)
	if err != nil {
		t.Fatalf("mustRoomName(%q): %v", raw, err)
	}
	return n
}

func mustInviteCode(t *testing.T, raw string) domain.InviteCode {
	t.Helper()
	c, err := domain.NewInviteCode(raw)
	if err != nil {
		t.Fatalf("mustInviteCode(%q): %v", raw, err)
	}
	return c
}

type RoomBuilder struct {
	t         *testing.T
	id        uuid.UUID
	ownerID   uuid.UUID
	name      string
	createdAt time.Time
}

func NewRoomBuilder(t *testing.T) *RoomBuilder {
	t.Helper()
	return &RoomBuilder{
		t:         t,
		id:        uuid.New(),
		ownerID:   uuid.New(),
		name:      "test-room",
		createdAt: defaultBuilderTime,
	}
}

func (b *RoomBuilder) WithID(id uuid.UUID) *RoomBuilder    { b.id = id; return b }
func (b *RoomBuilder) WithOwner(id uuid.UUID) *RoomBuilder { b.ownerID = id; return b }
func (b *RoomBuilder) WithName(name string) *RoomBuilder   { b.name = name; return b }
func (b *RoomBuilder) CreatedAt(t time.Time) *RoomBuilder  { b.createdAt = t; return b }

func (b *RoomBuilder) Build() *domain.Room {
	b.t.Helper()
	r, err := domain.NewRoom(
		mustRoomID(b.t, b.id),
		mustUserID(b.t, b.ownerID),
		mustRoomName(b.t, b.name),
		b.createdAt,
	)
	if err != nil {
		b.t.Fatalf("RoomBuilder.Build: %v", err)
	}
	return r
}

type MembershipBuilder struct {
	t        *testing.T
	roomID   uuid.UUID
	userID   uuid.UUID
	role     domain.Role
	joinedAt time.Time
}

func NewMembershipBuilder(t *testing.T) *MembershipBuilder {
	t.Helper()
	return &MembershipBuilder{
		t:        t,
		roomID:   uuid.New(),
		userID:   uuid.New(),
		role:     domain.RoleMember,
		joinedAt: defaultBuilderTime,
	}
}

func (b *MembershipBuilder) WithRoom(id uuid.UUID) *MembershipBuilder  { b.roomID = id; return b }
func (b *MembershipBuilder) WithUser(id uuid.UUID) *MembershipBuilder  { b.userID = id; return b }
func (b *MembershipBuilder) WithRole(r domain.Role) *MembershipBuilder { b.role = r; return b }
func (b *MembershipBuilder) JoinedAt(t time.Time) *MembershipBuilder   { b.joinedAt = t; return b }
func (b *MembershipBuilder) Owner() *MembershipBuilder                 { b.role = domain.RoleOwner; return b }
func (b *MembershipBuilder) Admin() *MembershipBuilder                 { b.role = domain.RoleAdmin; return b }
func (b *MembershipBuilder) Member() *MembershipBuilder                { b.role = domain.RoleMember; return b }

func (b *MembershipBuilder) Build() *domain.Membership {
	b.t.Helper()
	m, err := domain.NewMembership(
		mustRoomID(b.t, b.roomID),
		mustUserID(b.t, b.userID),
		b.role,
		b.joinedAt,
	)
	if err != nil {
		b.t.Fatalf("MembershipBuilder.Build: %v", err)
	}
	return m
}

type InviteBuilder struct {
	t         *testing.T
	id        uuid.UUID
	roomID    uuid.UUID
	code      string
	createdBy uuid.UUID
	createdAt time.Time
	revokedAt time.Time
}

func NewInviteBuilder(t *testing.T) *InviteBuilder {
	t.Helper()
	return &InviteBuilder{
		t:         t,
		id:        uuid.New(),
		roomID:    uuid.New(),
		code:      "ABCDEFGH",
		createdBy: uuid.New(),
		createdAt: defaultBuilderTime,
	}
}

func (b *InviteBuilder) WithID(id uuid.UUID) *InviteBuilder        { b.id = id; return b }
func (b *InviteBuilder) WithRoom(id uuid.UUID) *InviteBuilder      { b.roomID = id; return b }
func (b *InviteBuilder) WithCode(code string) *InviteBuilder       { b.code = code; return b }
func (b *InviteBuilder) WithCreatedBy(id uuid.UUID) *InviteBuilder { b.createdBy = id; return b }
func (b *InviteBuilder) CreatedAt(t time.Time) *InviteBuilder      { b.createdAt = t; return b }
func (b *InviteBuilder) Revoked(now time.Time) *InviteBuilder      { b.revokedAt = now; return b }

func (b *InviteBuilder) Build() *domain.Invite {
	b.t.Helper()
	inv, err := domain.ReconstructInvite(
		mustInviteID(b.t, b.id),
		mustRoomID(b.t, b.roomID),
		mustInviteCode(b.t, b.code),
		mustUserID(b.t, b.createdBy),
		b.createdAt,
		b.revokedAt,
	)
	if err != nil {
		b.t.Fatalf("InviteBuilder.Build: %v", err)
	}
	return inv
}
