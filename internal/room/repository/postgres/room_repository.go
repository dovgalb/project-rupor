package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/repository/postgres/db"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

// roomMembersOneOwnerPerRoomIdx — partial unique index, гарантирующий ровно одного owner на комнату.
// Срабатывание означает баг логики: usecase вызвал SaveWithOwner для уже существующей комнаты.
const roomMembersOneOwnerPerRoomIdx = "room_members_one_owner_per_room"

type RoomRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewRoomRepository(pool *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{pool: pool, q: db.New(pool)}
}

func (r *RoomRepository) SaveWithOwner(
	ctx context.Context,
	room *domain.Room,
	owner *domain.Membership,
) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("postgres: room save: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)
	if err := qtx.InsertRoom(ctx, domainToInsertRoomParams(room)); err != nil {
		return fmt.Errorf("postgres: room save: insert room: %w", err)
	}
	if err := qtx.InsertRoomMember(ctx, domainToInsertMembershipParams(owner)); err != nil {
		if isUniqueViolation(err, roomMembersOneOwnerPerRoomIdx) {
			return fmt.Errorf("postgres: room save: owner already exists: %w", err)
		}
		return fmt.Errorf("postgres: room save: insert owner: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: room save: commit: %w", err)
	}
	return nil
}

func (r *RoomRepository) FindByID(ctx context.Context, id domain.RoomID) (*domain.Room, error) {
	row, err := r.q.GetRoomByID(ctx, id.UUID())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRoomNotFound
		}
		return nil, fmt.Errorf("postgres: get room by id: %w", err)
	}
	return roomRowToDomain(row)
}

func (r *RoomRepository) ListByMember(
	ctx context.Context,
	userID domain.UserID,
) ([]usecase.RoomWithRole, error) {
	rows, err := r.q.ListRoomsByMember(ctx, userID.UUID())
	if err != nil {
		return nil, fmt.Errorf("postgres: list rooms by member: %w", err)
	}
	out := make([]usecase.RoomWithRole, 0, len(rows))
	for _, row := range rows {
		room, role, mErr := roomWithRoleRowToDomain(row)
		if mErr != nil {
			return nil, mErr
		}
		out = append(out, usecase.RoomWithRole{Room: room, Role: role})
	}
	return out, nil
}

func (r *RoomRepository) Delete(ctx context.Context, id domain.RoomID) error {
	affected, err := r.q.DeleteRoomByID(ctx, id.UUID())
	if err != nil {
		return fmt.Errorf("postgres: delete room: %w", err)
	}
	if affected == 0 {
		return domain.ErrRoomNotFound
	}
	return nil
}
