package postgres

import (
	"fmt"
	"time"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres/db"
)

func userRowToDomain(row db.User) (*domain.User, error) {
	id, err := domain.NewUserID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("user row: id: %w", err)
	}
	email, err := domain.NewEmail(row.Email)
	if err != nil {
		return nil, fmt.Errorf("user row: email: %w", err)
	}
	username, err := domain.NewUsername(row.Username)
	if err != nil {
		return nil, fmt.Errorf("user row: username: %w", err)
	}
	hash, err := domain.NewPasswordHash(row.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("user row: password_hash: %w", err)
	}
	return domain.ReconstructUser(id, email, username, hash, row.CreatedAt)
}

func domainToInsertUserParams(u *domain.User) db.InsertUserParams {
	return db.InsertUserParams{
		ID:           u.ID().UUID(),
		Email:        u.Email().String(),
		PasswordHash: u.PasswordHash().String(),
		Username:     u.Username().String(),
		CreatedAt:    u.CreatedAt(),
	}
}

func refreshTokenRowToDomain(row db.RefreshToken) (*domain.RefreshToken, error) {
	id, err := domain.NewRefreshTokenID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("refresh_token row: id: %w", err)
	}
	userID, err := domain.NewUserID(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("refresh_token row: user_id: %w", err)
	}
	hash, err := domain.NewTokenHash(row.TokenHash)
	if err != nil {
		return nil, fmt.Errorf("refresh_token row: token_hash: %w", err)
	}
	var revokedAt time.Time
	if row.RevokedAt.Valid {
		revokedAt = row.RevokedAt.Time
	}
	return domain.ReconstructRefreshToken(
		id,
		userID,
		hash,
		row.ExpiresAt,
		row.CreatedAt,
		revokedAt,
	)
}

func domainToInsertRefreshTokenParams(t *domain.RefreshToken) db.InsertRefreshTokenParams {
	bytes := t.TokenHash().Bytes()
	return db.InsertRefreshTokenParams{
		ID:        t.ID().UUID(),
		UserID:    t.UserID().UUID(),
		TokenHash: bytes[:],
		ExpiresAt: t.ExpiresAt(),
		CreatedAt: t.CreatedAt(),
	}
}
