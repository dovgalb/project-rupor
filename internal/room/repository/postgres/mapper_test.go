package postgres

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/repository/postgres/db"
)

// ---------- mapper test helpers ----------

func mustRoomID(t *testing.T, raw uuid.UUID) domain.RoomID {
	t.Helper()
	id, err := domain.NewRoomID(raw)
	if err != nil {
		t.Fatalf("NewRoomID: %v", err)
	}
	return id
}

func mustUserID(t *testing.T, raw uuid.UUID) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("NewUserID: %v", err)
	}
	return id
}

func mustInviteID(t *testing.T, raw uuid.UUID) domain.InviteID {
	t.Helper()
	id, err := domain.NewInviteID(raw)
	if err != nil {
		t.Fatalf("NewInviteID: %v", err)
	}
	return id
}

func mustRoomName(t *testing.T, raw string) domain.RoomName {
	t.Helper()
	n, err := domain.NewRoomName(raw)
	if err != nil {
		t.Fatalf("NewRoomName: %v", err)
	}
	return n
}

func mustInviteCode(t *testing.T, raw string) domain.InviteCode {
	t.Helper()
	c, err := domain.NewInviteCode(raw)
	if err != nil {
		t.Fatalf("NewInviteCode: %v", err)
	}
	return c
}

// ---------- Round-trip tests ----------

func TestRoomMapper_RoundTrip_AllFieldsPreserved(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	src, err := domain.NewRoom(
		mustRoomID(t, uuid.New()),
		mustUserID(t, uuid.New()),
		mustRoomName(t, "general"),
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewRoom: %v", err)
	}

	params := domainToInsertRoomParams(src)
	row := db.Room(params)

	got, err := roomRowToDomain(row)
	if err != nil {
		t.Fatalf("roomRowToDomain: %v", err)
	}
	if got.ID() != src.ID() {
		t.Fatalf("ID mismatch")
	}
	if got.OwnerID() != src.OwnerID() {
		t.Fatalf("OwnerID mismatch")
	}
	if got.Name() != src.Name() {
		t.Fatalf("Name mismatch")
	}
	if !got.CreatedAt().Equal(src.CreatedAt()) {
		t.Fatalf("CreatedAt mismatch")
	}
}

func TestMembershipMapper_RoundTrip_AllRoles(t *testing.T) {
	t.Parallel()

	joinedAt := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	roles := []domain.Role{domain.RoleOwner, domain.RoleAdmin, domain.RoleMember}

	for _, role := range roles {
		role := role
		t.Run(role.String(), func(t *testing.T) {
			t.Parallel()
			src, err := domain.NewMembership(
				mustRoomID(t, uuid.New()),
				mustUserID(t, uuid.New()),
				role,
				joinedAt,
			)
			if err != nil {
				t.Fatalf("NewMembership: %v", err)
			}
			params := domainToInsertMembershipParams(src)
			row := db.RoomMember(params)
			got, err := membershipRowToDomain(row)
			if err != nil {
				t.Fatalf("membershipRowToDomain: %v", err)
			}
			if got.RoomID() != src.RoomID() {
				t.Fatalf("RoomID mismatch")
			}
			if got.UserID() != src.UserID() {
				t.Fatalf("UserID mismatch")
			}
			if got.Role() != src.Role() {
				t.Fatalf("Role mismatch: got %v, want %v", got.Role(), src.Role())
			}
			if !got.JoinedAt().Equal(src.JoinedAt()) {
				t.Fatalf("JoinedAt mismatch")
			}
		})
	}
}

func TestInviteMapper_RoundTrip_Active(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	src, err := domain.NewInvite(
		mustInviteID(t, uuid.New()),
		mustRoomID(t, uuid.New()),
		mustInviteCode(t, "ABCDEFGH"),
		mustUserID(t, uuid.New()),
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewInvite: %v", err)
	}

	params := domainToInsertInviteParams(src)
	row := db.Invite{
		ID:        params.ID,
		RoomID:    params.RoomID,
		Code:      params.Code,
		CreatedBy: params.CreatedBy,
		CreatedAt: params.CreatedAt,
		RevokedAt: pgtype.Timestamptz{Valid: false},
	}

	got, err := inviteRowToDomain(row)
	if err != nil {
		t.Fatalf("inviteRowToDomain: %v", err)
	}
	if !got.IsActive() {
		t.Fatalf("expected active invite")
	}
	if got.ID() != src.ID() {
		t.Fatalf("ID mismatch")
	}
	if got.Code() != src.Code() {
		t.Fatalf("Code mismatch")
	}
}

func TestInviteMapper_RoundTrip_Revoked(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	revokedAt := createdAt.Add(time.Hour)
	src, err := domain.ReconstructInvite(
		mustInviteID(t, uuid.New()),
		mustRoomID(t, uuid.New()),
		mustInviteCode(t, "ABCDEFGH"),
		mustUserID(t, uuid.New()),
		createdAt,
		revokedAt,
	)
	if err != nil {
		t.Fatalf("ReconstructInvite: %v", err)
	}

	params := domainToInsertInviteParams(src)
	row := db.Invite{
		ID:        params.ID,
		RoomID:    params.RoomID,
		Code:      params.Code,
		CreatedBy: params.CreatedBy,
		CreatedAt: params.CreatedAt,
		RevokedAt: pgtype.Timestamptz{Time: revokedAt, Valid: true},
	}

	got, err := inviteRowToDomain(row)
	if err != nil {
		t.Fatalf("inviteRowToDomain: %v", err)
	}
	if !got.IsRevoked() {
		t.Fatalf("expected revoked invite")
	}
	if !got.RevokedAt().Equal(revokedAt) {
		t.Fatalf("RevokedAt mismatch: got %v, want %v", got.RevokedAt(), revokedAt)
	}
}

// ---------- Base32CodeGen test ----------

func TestBase32CodeGen_NewProducesValidCode(t *testing.T) {
	t.Parallel()

	// Заранее заданный поток байт → детерминированный код.
	buf := bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7})
	gen := NewBase32CodeGen(buf)

	code, err := gen.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(code.String()) != inviteCodeLength {
		t.Fatalf("code length = %d, want %d", len(code.String()), inviteCodeLength)
	}
	// 8 байт ровно расходуются — повторный вызов на пустом ридере должен упасть.
	if _, err := gen.New(); err == nil {
		t.Fatalf("expected error on exhausted reader, got nil")
	}
}
