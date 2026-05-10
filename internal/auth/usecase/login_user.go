package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

const refreshTokenRawSize = 32

type LoginUserInput struct {
	Email    string
	Password string
}

type LoginUserOutput struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type LoginUser struct {
	users      UserRepository
	refresh    RefreshTokenRepository
	hasher     PasswordHasher
	issuer     TokenIssuer
	clock      Clock
	uuids      UUIDGenerator
	rand       RandomBytes
	refreshTTL time.Duration
	dummyHash  domain.PasswordHash
}

func NewLoginUser(
	users UserRepository,
	refresh RefreshTokenRepository,
	hasher PasswordHasher,
	issuer TokenIssuer,
	clock Clock,
	uuids UUIDGenerator,
	rand RandomBytes,
	refreshTTL time.Duration,
	dummyHash domain.PasswordHash,
) *LoginUser {
	return &LoginUser{
		users:      users,
		refresh:    refresh,
		hasher:     hasher,
		issuer:     issuer,
		clock:      clock,
		uuids:      uuids,
		rand:       rand,
		refreshTTL: refreshTTL,
		dummyHash:  dummyHash,
	}
}

func (u *LoginUser) Execute(ctx context.Context, in LoginUserInput) (LoginUserOutput, error) {
	email, err := domain.NewEmail(in.Email)
	if err != nil {
		return LoginUserOutput{}, domain.ErrInvalidCredentials
	}
	password, err := domain.NewPassword(in.Password)
	if err != nil {
		return LoginUserOutput{}, domain.ErrInvalidCredentials
	}

	user, err := u.users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return LoginUserOutput{}, fmt.Errorf("usecase: login: find user: %w", err)
	}

	hashToVerify := u.dummyHash
	userFound := user != nil
	if userFound {
		hashToVerify = user.PasswordHash()
	}
	if vErr := u.hasher.Verify(hashToVerify, password); vErr != nil {
		return LoginUserOutput{}, domain.ErrInvalidCredentials
	}
	if !userFound {
		return LoginUserOutput{}, domain.ErrInvalidCredentials
	}

	now := u.clock.Now()

	accessToken, accessExp, err := u.issuer.IssueAccess(user.ID(), now)
	if err != nil {
		return LoginUserOutput{}, fmt.Errorf("usecase: login: issue access: %w", err)
	}

	raw, err := u.rand.Read(refreshTokenRawSize)
	if err != nil {
		return LoginUserOutput{}, fmt.Errorf("usecase: login: rand: %w", err)
	}

	digest := sha256.Sum256(raw)
	tokenHash, err := domain.NewTokenHash(digest[:])
	if err != nil {
		return LoginUserOutput{}, fmt.Errorf("usecase: login: token hash: %w", err)
	}

	rtID, err := domain.NewRefreshTokenID(u.uuids.New())
	if err != nil {
		return LoginUserOutput{}, fmt.Errorf("usecase: login: refresh id: %w", err)
	}

	refreshExp := now.Add(u.refreshTTL)
	rt, err := domain.NewRefreshToken(rtID, user.ID(), tokenHash, refreshExp, now)
	if err != nil {
		return LoginUserOutput{}, fmt.Errorf("usecase: login: new refresh: %w", err)
	}

	if err := u.refresh.Save(ctx, rt); err != nil {
		return LoginUserOutput{}, fmt.Errorf("usecase: login: save refresh: %w", err)
	}

	return LoginUserOutput{
		AccessToken:      accessToken,
		RefreshToken:     base64.RawURLEncoding.EncodeToString(raw),
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}, nil
}
