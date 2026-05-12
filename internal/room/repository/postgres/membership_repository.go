package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/repository/postgres/db"
)

const roomMembersPKey = "room_members_pkey"

type MembershipRepository struct {
	q *db.Queries
}

func NewMembershipRepository(pool *pgxpool.Pool) *MembershipRepository {
	return &MembershipRepository{q: db.New(pool)}
}

func (r *MembershipRepository) FindByPair(
	ctx context.Context,
	roomID domain.RoomID,
	userID domain.UserID,
) (*domain.Membership, error) {
	row, err := r.q.GetRoomMember(ctx, db.GetRoomMemberParams{
		RoomID: roomID.UUID(),
		UserID: userID.UUID(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotMember
		}
		return nil, fmt.Errorf("postgres: get room member: %w", err)
	}
	return membershipRowToDomain(row)
}

func (r *MembershipRepository) ListByRoom(
	ctx context.Context,
	roomID domain.RoomID,
) ([]*domain.Membership, error) {
	rows, err := r.q.ListRoomMembers(ctx, roomID.UUID())
	if err != nil {
		return nil, fmt.Errorf("postgres: list room members: %w", err)
	}
	out := make([]*domain.Membership, 0, len(rows))
	for _, row := range rows {
		m, mErr := membershipRowToDomain(row)
		if mErr != nil {
			return nil, mErr
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *MembershipRepository) Add(ctx context.Context, m *domain.Membership) error {
	if err := r.q.InsertRoomMember(ctx, domainToInsertMembershipParams(m)); err != nil {
		if isUniqueViolation(err, roomMembersPKey) {
			return domain.ErrAlreadyMember
		}
		if isUniqueViolation(err, roomMembersOneOwnerPerRoomIdx) {
			return fmt.Errorf("postgres: insert membership: owner already exists: %w", err)
		}
		return fmt.Errorf("postgres: insert membership: %w", err)
	}
	return nil
}
