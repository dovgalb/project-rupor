package jwt_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/repository/jwt"
)

const (
	accessTTL = 15 * time.Minute
)

var (
	secret      = []byte("test-secret-1234567890")
	wrongSecret = []byte("other-secret-0987654321")
)

func newIssuer(t *testing.T) *jwt.TokenIssuer {
	t.Helper()
	return jwt.NewTokenIssuer(secret, accessTTL)
}

func mustUserID(t *testing.T) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(uuid.New())
	if err != nil {
		t.Fatalf("NewUserID: %v", err)
	}
	return id
}

func TestTokenIssuer_RoundTrip(t *testing.T) {
	t.Parallel()

	issuer := newIssuer(t)
	uid := mustUserID(t)
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	token, expiresAt, err := issuer.IssueAccess(uid, now)
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	if !expiresAt.Equal(now.Add(accessTTL)) {
		t.Fatalf("expiresAt mismatch")
	}

	got, err := issuer.VerifyAccess(token, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("VerifyAccess: %v", err)
	}
	if got != uid {
		t.Fatalf("UserID mismatch")
	}
}

func TestTokenIssuer_VerifyAccess_RejectsAlgNone(t *testing.T) {
	t.Parallel()

	issuer := newIssuer(t)
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + uuid.New().String() + `","exp":99999999999}`))
	token := strings.Join([]string{header, payload, ""}, ".")

	_, err := issuer.VerifyAccess(token, now)
	if !errors.Is(err, domain.ErrAccessTokenInvalid) {
		t.Fatalf("got %v, want ErrAccessTokenInvalid", err)
	}
}

func TestTokenIssuer_VerifyAccess_RejectsWrongSecret(t *testing.T) {
	t.Parallel()

	issuer := jwt.NewTokenIssuer(secret, accessTTL)
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	claims := jwtv5.RegisteredClaims{
		Subject:   uuid.New().String(),
		IssuedAt:  jwtv5.NewNumericDate(now),
		ExpiresAt: jwtv5.NewNumericDate(now.Add(accessTTL)),
	}
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	signed, err := tok.SignedString(wrongSecret)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	_, err = issuer.VerifyAccess(signed, now)
	if !errors.Is(err, domain.ErrAccessTokenInvalid) {
		t.Fatalf("got %v, want ErrAccessTokenInvalid", err)
	}
}

func TestTokenIssuer_VerifyAccess_RejectsExpired(t *testing.T) {
	t.Parallel()

	issuer := newIssuer(t)
	uid := mustUserID(t)
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	token, _, err := issuer.IssueAccess(uid, now)
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}

	_, err = issuer.VerifyAccess(token, now.Add(accessTTL+time.Minute))
	if !errors.Is(err, domain.ErrAccessTokenExpired) {
		t.Fatalf("got %v, want ErrAccessTokenExpired", err)
	}
}

func TestTokenIssuer_VerifyAccess_RejectsBadSub(t *testing.T) {
	t.Parallel()

	issuer := newIssuer(t)
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	claims := jwtv5.RegisteredClaims{
		Subject:   "not-a-uuid",
		IssuedAt:  jwtv5.NewNumericDate(now),
		ExpiresAt: jwtv5.NewNumericDate(now.Add(accessTTL)),
	}
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	signed, err := tok.SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	_, err = issuer.VerifyAccess(signed, now)
	if !errors.Is(err, domain.ErrAccessTokenInvalid) {
		t.Fatalf("got %v, want ErrAccessTokenInvalid", err)
	}
}

func TestTokenIssuer_VerifyAccess_RejectsTampered(t *testing.T) {
	t.Parallel()

	issuer := newIssuer(t)
	uid := mustUserID(t)
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	token, _, err := issuer.IssueAccess(uid, now)
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token structure")
	}
	tamperedPayload := base64.RawURLEncoding.EncodeToString([]byte(
		`{"sub":"` + uuid.New().String() + `","exp":99999999999}`,
	))
	tampered := strings.Join([]string{parts[0], tamperedPayload, parts[2]}, ".")

	_, err = issuer.VerifyAccess(tampered, now)
	if !errors.Is(err, domain.ErrAccessTokenInvalid) {
		t.Fatalf("got %v, want ErrAccessTokenInvalid", err)
	}
}
