package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres/db"
)

const refreshTokenHashKey = "refresh_tokens_token_hash_key"

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool, q: db.New(pool)}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, t *domain.RefreshToken) error {
	if err := r.q.InsertRefreshToken(ctx, domainToInsertRefreshTokenParams(t)); err != nil {
		if isUniqueViolation(err, refreshTokenHashKey) {
			return fmt.Errorf("postgres: insert refresh: hash collision: %w", err)
		}
		return fmt.Errorf("postgres: insert refresh: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, h domain.TokenHash) (*domain.RefreshToken, error) {
	hashBytes := h.Bytes()
	row, err := r.q.GetRefreshTokenByHash(ctx, hashBytes[:])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("postgres: get refresh by hash: %w", err)
	}
	return refreshTokenRowToDomain(row)
}

func (r *RefreshTokenRepository) Rotate(
	ctx context.Context,
	oldHash domain.TokenHash,
	newToken *domain.RefreshToken,
	revokedAt time.Time,
) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)
	oldBytes := oldHash.Bytes()
	affected, err := qtx.RevokeRefreshTokenByHash(ctx, db.RevokeRefreshTokenByHashParams{
		TokenHash: oldBytes[:],
		RevokedAt: pgtype.Timestamptz{Time: revokedAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("postgres: revoke old refresh: %w", err)
	}
	if affected == 0 {
		return domain.ErrRefreshTokenRevoked
	}

	if err := qtx.InsertRefreshToken(ctx, domainToInsertRefreshTokenParams(newToken)); err != nil {
		if isUniqueViolation(err, refreshTokenHashKey) {
			return fmt.Errorf("postgres: insert new refresh: hash collision: %w", err)
		}
		return fmt.Errorf("postgres: insert new refresh: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit rotate: %w", err)
	}
	return nil
}
