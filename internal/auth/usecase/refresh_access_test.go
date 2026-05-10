package usecase_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

const refreshTestTTL = 24 * time.Hour

type refreshSUT struct {
	uc          *usecase.RefreshAccess
	refreshRepo *fakeRefreshRepo
	issuer      *fakeIssuer
	rand        *fixedRand
	now         time.Time
	user        *domain.User
	oldRaw      []byte
	oldEncoded  string
	oldToken    *domain.RefreshToken
	newRaw      []byte
}

func newRefreshSUT(t *testing.T) *refreshSUT {
	t.Helper()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	user := mustUser(t, "user@example.com", "user_1", "hashed:correct", now.Add(-time.Hour))

	oldRaw := bytes.Repeat([]byte{0xAA}, 32)
	oldDigest := sha256.Sum256(oldRaw)
	oldHash := mustTokenHash(t, oldDigest[:])
	oldToken, err := domain.NewRefreshToken(
		mustRefreshTokenID(t, uuid.New()),
		user.ID(),
		oldHash,
		now.Add(refreshTestTTL),
		now.Add(-time.Minute),
	)
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}

	refreshRepo := newFakeRefreshRepo()
	refreshRepo.byHash[oldHash.Bytes()] = oldToken

	newRaw := bytes.Repeat([]byte{0xBB}, 32)
	issuer := &fakeIssuer{accessTTL: 15 * time.Minute}
	rand := &fixedRand{next: [][]byte{newRaw}}

	uc := usecase.NewRefreshAccess(
		refreshRepo,
		issuer,
		&fixedClock{now: now},
		&fixedUUID{next: []uuid.UUID{uuid.New()}},
		rand,
		refreshTestTTL,
	)
	return &refreshSUT{
		uc: uc, refreshRepo: refreshRepo, issuer: issuer, rand: rand,
		now: now, user: user,
		oldRaw: oldRaw, oldEncoded: base64.RawURLEncoding.EncodeToString(oldRaw),
		oldToken: oldToken, newRaw: newRaw,
	}
}

func TestRefreshAccess_Success(t *testing.T) {
	t.Parallel()

	sut := newRefreshSUT(t)
	out, err := sut.uc.Execute(context.Background(), usecase.RefreshAccessInput{RefreshToken: sut.oldEncoded})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.AccessToken != "access-"+sut.user.ID().String() {
		t.Fatalf("AccessToken = %q", out.AccessToken)
	}
	if len(sut.refreshRepo.rotateOld) != 1 {
		t.Fatalf("expected 1 Rotate, got %d", len(sut.refreshRepo.rotateOld))
	}
	expectedOld := sha256.Sum256(sut.oldRaw)
	if sut.refreshRepo.rotateOld[0].Bytes() != expectedOld {
		t.Fatalf("rotated old hash mismatch")
	}
	if sut.refreshRepo.rotateNew[0].UserID() != sut.user.ID() {
		t.Fatalf("rotated new userID mismatch")
	}
}

func TestRefreshAccess_NotFound(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		setup func(*refreshSUT)
	}{
		{"невалидный base64", "!!!!", func(*refreshSUT) {}},
		{"длина после декода != 32", base64.RawURLEncoding.EncodeToString([]byte("short")), func(*refreshSUT) {}},
		{"FindByHash → ErrRefreshTokenNotFound", "", func(s *refreshSUT) {
			s.refreshRepo.byHash = map[[32]byte]*domain.RefreshToken{}
		}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sut := newRefreshSUT(t)
			tc.setup(sut)
			input := tc.input
			if input == "" {
				input = sut.oldEncoded
			}
			_, err := sut.uc.Execute(context.Background(), usecase.RefreshAccessInput{RefreshToken: input})
			if !errors.Is(err, domain.ErrRefreshTokenNotFound) {
				t.Fatalf("got %v, want ErrRefreshTokenNotFound", err)
			}
		})
	}
}

func TestRefreshAccess_Revoked(t *testing.T) {
	t.Parallel()

	sut := newRefreshSUT(t)
	if err := sut.oldToken.Revoke(sut.now.Add(-time.Second)); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	_, err := sut.uc.Execute(context.Background(), usecase.RefreshAccessInput{RefreshToken: sut.oldEncoded})
	if !errors.Is(err, domain.ErrRefreshTokenRevoked) {
		t.Fatalf("got %v, want ErrRefreshTokenRevoked", err)
	}
}

func TestRefreshAccess_Expired(t *testing.T) {
	t.Parallel()

	sut := newRefreshSUT(t)
	expiredToken, err := domain.NewRefreshToken(
		mustRefreshTokenID(t, uuid.New()),
		sut.user.ID(),
		sut.oldToken.TokenHash(),
		sut.now.Add(-time.Hour),
		sut.now.Add(-2*time.Hour),
	)
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	sut.refreshRepo.byHash[sut.oldToken.TokenHash().Bytes()] = expiredToken

	_, err = sut.uc.Execute(context.Background(), usecase.RefreshAccessInput{RefreshToken: sut.oldEncoded})
	if !errors.Is(err, domain.ErrRefreshTokenExpired) {
		t.Fatalf("got %v, want ErrRefreshTokenExpired", err)
	}
}

func TestRefreshAccess_RaceReturnsRevoked(t *testing.T) {
	t.Parallel()

	sut := newRefreshSUT(t)
	sut.refreshRepo.rotateErr = domain.ErrRefreshTokenRevoked

	_, err := sut.uc.Execute(context.Background(), usecase.RefreshAccessInput{RefreshToken: sut.oldEncoded})
	if !errors.Is(err, domain.ErrRefreshTokenRevoked) {
		t.Fatalf("got %v, want ErrRefreshTokenRevoked", err)
	}
}

func TestRefreshAccess_TokenIssuerFails(t *testing.T) {
	t.Parallel()

	sut := newRefreshSUT(t)
	sut.issuer.issueErr = errors.New("issuer boom")

	_, err := sut.uc.Execute(context.Background(), usecase.RefreshAccessInput{RefreshToken: sut.oldEncoded})
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(sut.refreshRepo.rotateOld) != 0 {
		t.Fatalf("Rotate must not be called on issuer failure")
	}
}
