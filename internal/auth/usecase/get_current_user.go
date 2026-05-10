package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

type GetCurrentUserInput struct {
	UserID string
}

type GetCurrentUserOutput struct {
	UserID    string
	Email     string
	Username  string
	CreatedAt time.Time
}

type GetCurrentUser struct {
	users UserRepository
}

func NewGetCurrentUser(users UserRepository) *GetCurrentUser {
	return &GetCurrentUser{users: users}
}

func (u *GetCurrentUser) Execute(ctx context.Context, in GetCurrentUserInput) (GetCurrentUserOutput, error) {
	raw, err := uuid.Parse(in.UserID)
	if err != nil {
		return GetCurrentUserOutput{}, domain.ErrInvalidUserID
	}
	id, err := domain.NewUserID(raw)
	if err != nil {
		return GetCurrentUserOutput{}, domain.ErrInvalidUserID
	}
	user, err := u.users.FindByID(ctx, id)
	if err != nil {
		return GetCurrentUserOutput{}, err
	}
	return GetCurrentUserOutput{
		UserID:    user.ID().String(),
		Email:     user.Email().String(),
		Username:  user.Username().String(),
		CreatedAt: user.CreatedAt(),
	}, nil
}
