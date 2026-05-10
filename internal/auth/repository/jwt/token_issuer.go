package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

type TokenIssuer struct {
	secret    []byte
	accessTTL time.Duration
}

func NewTokenIssuer(secret []byte, accessTTL time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: secret, accessTTL: accessTTL}
}

func (i *TokenIssuer) IssueAccess(userID domain.UserID, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(i.accessTTL)
	claims := jwtv5.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwtv5.NewNumericDate(now),
		ExpiresAt: jwtv5.NewNumericDate(expiresAt),
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	signed, err := token.SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, expiresAt, nil
}

func (i *TokenIssuer) VerifyAccess(token string, now time.Time) (domain.UserID, error) {
	parser := jwtv5.NewParser(
		jwtv5.WithValidMethods([]string{"HS256"}),
		jwtv5.WithTimeFunc(func() time.Time { return now }),
	)
	var claims jwtv5.RegisteredClaims
	_, err := parser.ParseWithClaims(token, &claims, func(t *jwtv5.Token) (any, error) {
		return i.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwtv5.ErrTokenExpired) {
			return domain.UserID{}, domain.ErrAccessTokenExpired
		}
		return domain.UserID{}, fmt.Errorf("%w: %v", domain.ErrAccessTokenInvalid, err)
	}
	raw, err := uuid.Parse(claims.Subject)
	if err != nil {
		return domain.UserID{}, domain.ErrAccessTokenInvalid
	}
	id, err := domain.NewUserID(raw)
	if err != nil {
		return domain.UserID{}, domain.ErrAccessTokenInvalid
	}
	return id, nil
}
