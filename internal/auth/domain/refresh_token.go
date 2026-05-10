package domain

import "time"

type RefreshToken struct {
	id        RefreshTokenID
	userID    UserID
	tokenHash TokenHash
	expiresAt time.Time
	createdAt time.Time
	revokedAt time.Time // zero value = активный
}

func NewRefreshToken(id RefreshTokenID, userID UserID, hash TokenHash, expiresAt, createdAt time.Time) (*RefreshToken, error) {
	if userID.IsZero() {
		return nil, ErrInvalidUserID
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidCreatedAt
	}
	if !expiresAt.After(createdAt) {
		return nil, ErrInvalidRefreshTokenExpiration
	}
	return &RefreshToken{
		id:        id,
		userID:    userID,
		tokenHash: hash,
		expiresAt: expiresAt,
		createdAt: createdAt,
	}, nil
}

func ReconstructRefreshToken(id RefreshTokenID, userID UserID, hash TokenHash, expiresAt, createdAt, revokedAt time.Time) (*RefreshToken, error) {
	t, err := NewRefreshToken(id, userID, hash, expiresAt, createdAt)
	if err != nil {
		return nil, err
	}
	t.revokedAt = revokedAt
	return t, nil
}

func (t *RefreshToken) Revoke(now time.Time) error {
	if !t.revokedAt.IsZero() {
		return ErrRefreshTokenAlreadyRevoked
	}
	t.revokedAt = now
	return nil
}

func (t *RefreshToken) IsActive(now time.Time) bool {
	return t.revokedAt.IsZero() && now.Before(t.expiresAt)
}

func (t *RefreshToken) IsRevoked() bool { return !t.revokedAt.IsZero() }

func (t *RefreshToken) IsExpired(now time.Time) bool { return !now.Before(t.expiresAt) }

func (t *RefreshToken) ID() RefreshTokenID { return t.id }
func (t *RefreshToken) UserID() UserID     { return t.userID }
func (t *RefreshToken) TokenHash() TokenHash {
	return t.tokenHash
}
func (t *RefreshToken) ExpiresAt() time.Time { return t.expiresAt }
func (t *RefreshToken) CreatedAt() time.Time { return t.createdAt }
func (t *RefreshToken) RevokedAt() time.Time { return t.revokedAt }
