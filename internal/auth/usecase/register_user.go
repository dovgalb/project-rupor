package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

type RegisterUserInput struct {
	Email    string
	Username string
	Password string
}

type RegisterUserOutput struct {
	UserID    string
	Email     string
	Username  string
	CreatedAt time.Time
}

type RegisterUser struct {
	users  UserRepository
	hasher PasswordHasher
	clock  Clock
	uuids  UUIDGenerator
}

func NewRegisterUser(users UserRepository, hasher PasswordHasher, clock Clock, uuids UUIDGenerator) *RegisterUser {
	return &RegisterUser{users: users, hasher: hasher, clock: clock, uuids: uuids}
}

func (u *RegisterUser) Execute(ctx context.Context, in RegisterUserInput) (RegisterUserOutput, error) {
	email, err := domain.NewEmail(in.Email)
	if err != nil {
		return RegisterUserOutput{}, err
	}
	username, err := domain.NewUsername(in.Username)
	if err != nil {
		return RegisterUserOutput{}, err
	}
	password, err := domain.NewPassword(in.Password)
	if err != nil {
		return RegisterUserOutput{}, err
	}

	hash, err := u.hasher.Hash(password)
	if err != nil {
		return RegisterUserOutput{}, fmt.Errorf("usecase: register: hash: %w", err)
	}

	id, err := domain.NewUserID(u.uuids.New())
	if err != nil {
		return RegisterUserOutput{}, fmt.Errorf("usecase: register: user id: %w", err)
	}

	now := u.clock.Now()
	user, err := domain.NewUser(id, email, username, hash, now)
	if err != nil {
		return RegisterUserOutput{}, fmt.Errorf("usecase: register: new user: %w", err)
	}

	if err := u.users.Save(ctx, user); err != nil {
		return RegisterUserOutput{}, err
	}

	return RegisterUserOutput{
		UserID:    user.ID().String(),
		Email:     user.Email().String(),
		Username:  user.Username().String(),
		CreatedAt: user.CreatedAt(),
	}, nil
}
