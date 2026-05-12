package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/repository/postgres/db"
)

// Room mappers.

func roomRowToDomain(row db.Room) (*domain.Room, error) {
	id, err := domain.NewRoomID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("room row: id: %w", err)
	}
	ownerID, err := domain.NewUserID(row.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("room row: owner_id: %w", err)
	}
	name, err := domain.NewRoomName(row.Name)
	if err != nil {
		return nil, fmt.Errorf("room row: name: %w", err)
	}
	return domain.ReconstructRoom(id, ownerID, name, row.CreatedAt)
}

func domainToInsertRoomParams(r *domain.Room) db.InsertRoomParams {
	return db.InsertRoomParams{
		ID:        r.ID().UUID(),
		OwnerID:   r.OwnerID().UUID(),
		Name:      r.Name().String(),
		CreatedAt: r.CreatedAt(),
	}
}

// Membership mappers.

func membershipRowToDomain(row db.RoomMember) (*domain.Membership, error) {
	roomID, err := domain.NewRoomID(row.RoomID)
	if err != nil {
		return nil, fmt.Errorf("membership row: room_id: %w", err)
	}
	userID, err := domain.NewUserID(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("membership row: user_id: %w", err)
	}
	role, err := domain.ParseRole(row.Role)
	if err != nil {
		return nil, fmt.Errorf("membership row: role: %w", err)
	}
	return domain.ReconstructMembership(roomID, userID, role, row.JoinedAt)
}

func domainToInsertMembershipParams(m *domain.Membership) db.InsertRoomMemberParams {
	return db.InsertRoomMemberParams{
		RoomID:   m.RoomID().UUID(),
		UserID:   m.UserID().UUID(),
		Role:     m.Role().String(),
		JoinedAt: m.JoinedAt(),
	}
}

// Invite mappers.

func inviteRowToDomain(row db.Invite) (*domain.Invite, error) {
	id, err := domain.NewInviteID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invite row: id: %w", err)
	}
	roomID, err := domain.NewRoomID(row.RoomID)
	if err != nil {
		return nil, fmt.Errorf("invite row: room_id: %w", err)
	}
	code, err := domain.NewInviteCode(row.Code)
	if err != nil {
		return nil, fmt.Errorf("invite row: code: %w", err)
	}
	createdBy, err := domain.NewUserID(row.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("invite row: created_by: %w", err)
	}
	var revokedAt time.Time
	if row.RevokedAt.Valid {
		revokedAt = row.RevokedAt.Time
	}
	return domain.ReconstructInvite(id, roomID, code, createdBy, row.CreatedAt, revokedAt)
}

func domainToInsertInviteParams(inv *domain.Invite) db.InsertInviteParams {
	return db.InsertInviteParams{
		ID:        inv.ID().UUID(),
		RoomID:    inv.RoomID().UUID(),
		Code:      inv.Code().String(),
		CreatedBy: inv.CreatedBy().UUID(),
		CreatedAt: inv.CreatedAt(),
	}
}

// roomWithRoleRowToDomain маппит результат ListRoomsByMember в (Room, role string).
func roomWithRoleRowToDomain(row db.ListRoomsByMemberRow) (*domain.Room, domain.Role, error) {
	id, err := domain.NewRoomID(row.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("room+role row: id: %w", err)
	}
	ownerID, err := domain.NewUserID(row.OwnerID)
	if err != nil {
		return nil, 0, fmt.Errorf("room+role row: owner_id: %w", err)
	}
	name, err := domain.NewRoomName(row.Name)
	if err != nil {
		return nil, 0, fmt.Errorf("room+role row: name: %w", err)
	}
	role, err := domain.ParseRole(row.Role)
	if err != nil {
		return nil, 0, fmt.Errorf("room+role row: role: %w", err)
	}
	room, err := domain.ReconstructRoom(id, ownerID, name, row.CreatedAt)
	if err != nil {
		return nil, 0, err
	}
	return room, role, nil
}

// nullableTimestamptz конвертирует time.Time в pgtype.Timestamptz; zero → Valid:false.
func nullableTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}
