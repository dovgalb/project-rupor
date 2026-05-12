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

const invitesActiveCodeIdx = "invites_active_code"

type InviteRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewInviteRepository(pool *pgxpool.Pool) *InviteRepository {
	return &InviteRepository{pool: pool, q: db.New(pool)}
}

// RegenerateActive в одной транзакции отзывает активные инвайты комнаты и вставляет новый.
// На unique violation `invites_active_code` (partial unique) возвращает usecase.ErrInviteCodeCollision
// — usecase выполнит ретрай.
// invite.CreatedAt() используется как timestamp отзыва старых.
func (r *InviteRepository) RegenerateActive(ctx context.Context, invite *domain.Invite) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("postgres: invite regenerate: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)
	if _, err := qtx.RevokeActiveInvitesByRoom(ctx, db.RevokeActiveInvitesByRoomParams{
		RoomID:    invite.RoomID().UUID(),
		RevokedAt: nullableTimestamptz(invite.CreatedAt()),
	}); err != nil {
		return fmt.Errorf("postgres: invite regenerate: revoke active: %w", err)
	}
	if err := qtx.InsertInvite(ctx, domainToInsertInviteParams(invite)); err != nil {
		if isUniqueViolation(err, invitesActiveCodeIdx) {
			return usecase.ErrInviteCodeCollision
		}
		return fmt.Errorf("postgres: invite regenerate: insert: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: invite regenerate: commit: %w", err)
	}
	return nil
}

func (r *InviteRepository) FindActiveByCode(
	ctx context.Context,
	code domain.InviteCode,
) (*domain.Invite, error) {
	row, err := r.q.GetActiveInviteByCode(ctx, code.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInviteNotFound
		}
		return nil, fmt.Errorf("postgres: get active invite by code: %w", err)
	}
	return inviteRowToDomain(row)
}
