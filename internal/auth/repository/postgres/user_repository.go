package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres/db"
)

const (
	usersEmailKey    = "users_email_key"
	usersUsernameKey = "users_username_key"
)

type UserRepository struct {
	q *db.Queries
}

func NewUserRepository(q *db.Queries) *UserRepository {
	return &UserRepository{q: q}
}

func (r *UserRepository) Save(ctx context.Context, u *domain.User) error {
	if err := r.q.InsertUser(ctx, domainToInsertUserParams(u)); err != nil {
		if isUniqueViolation(err, usersEmailKey) {
			return domain.ErrEmailAlreadyTaken
		}
		if isUniqueViolation(err, usersUsernameKey) {
			return domain.ErrUsernameAlreadyTaken
		}
		return fmt.Errorf("postgres: insert user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	row, err := r.q.GetUserByID(ctx, id.UUID())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("postgres: get user by id: %w", err)
	}
	return userRowToDomain(row)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("postgres: get user by email: %w", err)
	}
	return userRowToDomain(row)
}
