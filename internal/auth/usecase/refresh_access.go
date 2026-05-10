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

type RefreshAccessInput struct {
	RefreshToken string
}

type RefreshAccessOutput = LoginUserOutput

type RefreshAccess struct {
	refresh    RefreshTokenRepository
	issuer     TokenIssuer
	clock      Clock
	uuids      UUIDGenerator
	rand       RandomBytes
	refreshTTL time.Duration
}

func NewRefreshAccess(
	refresh RefreshTokenRepository,
	issuer TokenIssuer,
	clock Clock,
	uuids UUIDGenerator,
	rand RandomBytes,
	refreshTTL time.Duration,
) *RefreshAccess {
	return &RefreshAccess{
		refresh:    refresh,
		issuer:     issuer,
		clock:      clock,
		uuids:      uuids,
		rand:       rand,
		refreshTTL: refreshTTL,
	}
}

func (u *RefreshAccess) Execute(ctx context.Context, in RefreshAccessInput) (RefreshAccessOutput, error) {
	raw, err := base64.RawURLEncoding.DecodeString(in.RefreshToken)
	if err != nil {
		return RefreshAccessOutput{}, domain.ErrRefreshTokenNotFound
	}
	if len(raw) != refreshTokenRawSize {
		return RefreshAccessOutput{}, domain.ErrRefreshTokenNotFound
	}

	digest := sha256.Sum256(raw)
	oldHash, err := domain.NewTokenHash(digest[:])
	if err != nil {
		return RefreshAccessOutput{}, fmt.Errorf("usecase: refresh: token hash: %w", err)
	}

	existing, err := u.refresh.FindByHash(ctx, oldHash)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return RefreshAccessOutput{}, err
		}
		return RefreshAccessOutput{}, fmt.Errorf("usecase: refresh: find: %w", err)
	}

	now := u.clock.Now()
	if existing.IsRevoked() {
		return RefreshAccessOutput{}, domain.ErrRefreshTokenRevoked
	}
	if existing.IsExpired(now) {
		return RefreshAccessOutput{}, domain.ErrRefreshTokenExpired
	}

	accessToken, accessExp, err := u.issuer.IssueAccess(existing.UserID(), now)
	if err != nil {
		return RefreshAccessOutput{}, fmt.Errorf("usecase: refresh: issue access: %w", err)
	}

	newRaw, err := u.rand.Read(refreshTokenRawSize)
	if err != nil {
		return RefreshAccessOutput{}, fmt.Errorf("usecase: refresh: rand: %w", err)
	}
	newDigest := sha256.Sum256(newRaw)
	newTokenHash, err := domain.NewTokenHash(newDigest[:])
	if err != nil {
		return RefreshAccessOutput{}, fmt.Errorf("usecase: refresh: new token hash: %w", err)
	}

	rtID, err := domain.NewRefreshTokenID(u.uuids.New())
	if err != nil {
		return RefreshAccessOutput{}, fmt.Errorf("usecase: refresh: refresh id: %w", err)
	}

	refreshExp := now.Add(u.refreshTTL)
	newRT, err := domain.NewRefreshToken(rtID, existing.UserID(), newTokenHash, refreshExp, now)
	if err != nil {
		return RefreshAccessOutput{}, fmt.Errorf("usecase: refresh: new refresh: %w", err)
	}

	if err := u.refresh.Rotate(ctx, oldHash, newRT, now); err != nil {
		if errors.Is(err, domain.ErrRefreshTokenRevoked) {
			return RefreshAccessOutput{}, err
		}
		return RefreshAccessOutput{}, fmt.Errorf("usecase: refresh: rotate: %w", err)
	}

	return RefreshAccessOutput{
		AccessToken:      accessToken,
		RefreshToken:     base64.RawURLEncoding.EncodeToString(newRaw),
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}, nil
}
