package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

type UserRepository interface {
	Save(ctx context.Context, u *domain.User) error
	FindByID(ctx context.Context, id domain.UserID) (*domain.User, error)
	FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error)
}

type RefreshTokenRepository interface {
	Save(ctx context.Context, t *domain.RefreshToken) error
	FindByHash(ctx context.Context, h domain.TokenHash) (*domain.RefreshToken, error)
	Rotate(ctx context.Context, oldHash domain.TokenHash, newToken *domain.RefreshToken, revokedAt time.Time) error
}

type PasswordHasher interface {
	Hash(p domain.Password) (domain.PasswordHash, error)
	Verify(h domain.PasswordHash, p domain.Password) error
}

type TokenIssuer interface {
	IssueAccess(userID domain.UserID, now time.Time) (token string, expiresAt time.Time, err error)
	VerifyAccess(token string, now time.Time) (domain.UserID, error)
}

type Clock interface {
	Now() time.Time
}

type UUIDGenerator interface {
	New() uuid.UUID
}

type RandomBytes interface {
	Read(n int) ([]byte, error)
}
