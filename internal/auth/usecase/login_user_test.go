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

const (
	loginAccessTTL  = 15 * time.Minute
	loginRefreshTTL = 24 * time.Hour
)

type loginSUT struct {
	uc         *usecase.LoginUser
	users      *fakeUserRepo
	refresh    *fakeRefreshRepo
	hasher     *fakeHasher
	issuer     *fakeIssuer
	clock      *fixedClock
	rand       *fixedRand
	now        time.Time
	user       *domain.User
	password   string
	rawRefresh []byte
}

func newLoginSUTFull(t *testing.T) *loginSUT {
	t.Helper()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	password := "correct-password"
	storedHash := mustPasswordHash(t, "hashed:"+password)
	user := mustUser(t, "user@example.com", "user_1", storedHash.String(), now.Add(-time.Hour))

	users := newFakeUserRepo()
	users.byEmail[user.Email().String()] = user
	users.byID[user.ID().String()] = user

	hasher := newFakeHasher()
	hasher.hashes[password] = storedHash

	refresh := newFakeRefreshRepo()
	issuer := &fakeIssuer{accessTTL: loginAccessTTL}
	clock := &fixedClock{now: now}
	rand := &fixedRand{next: [][]byte{bytes.Repeat([]byte{0xCC}, 32)}}

	uc := usecase.NewLoginUser(
		users,
		refresh,
		hasher,
		issuer,
		clock,
		&fixedUUID{next: []uuid.UUID{uuid.New()}},
		rand,
		loginRefreshTTL,
		mustPasswordHash(t, "$2a$10$dummyhashvalue"),
	)
	return &loginSUT{
		uc: uc, users: users, refresh: refresh, hasher: hasher, issuer: issuer,
		clock: clock, rand: rand, now: now, user: user, password: password,
		rawRefresh: bytes.Repeat([]byte{0xCC}, 32),
	}
}

func TestLoginUser_Success(t *testing.T) {
	t.Parallel()

	sut := newLoginSUTFull(t)
	out, err := sut.uc.Execute(context.Background(), usecase.LoginUserInput{
		Email:    sut.user.Email().String(),
		Password: sut.password,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.AccessToken != "access-"+sut.user.ID().String() {
		t.Fatalf("AccessToken = %q", out.AccessToken)
	}
	if !out.AccessExpiresAt.Equal(sut.now.Add(loginAccessTTL)) {
		t.Fatalf("AccessExpiresAt = %s", out.AccessExpiresAt)
	}
	if !out.RefreshExpiresAt.Equal(sut.now.Add(loginRefreshTTL)) {
		t.Fatalf("RefreshExpiresAt = %s", out.RefreshExpiresAt)
	}
	rawDecoded, err := base64.RawURLEncoding.DecodeString(out.RefreshToken)
	if err != nil {
		t.Fatalf("decode refresh: %v", err)
	}
	if !bytes.Equal(rawDecoded, sut.rawRefresh) {
		t.Fatalf("decoded refresh != raw")
	}
	if len(sut.refresh.saved) != 1 {
		t.Fatalf("expected 1 refresh saved")
	}
	expectedDigest := sha256.Sum256(sut.rawRefresh)
	if sut.refresh.saved[0].TokenHash().Bytes() != expectedDigest {
		t.Fatalf("saved hash != sha256(raw)")
	}
	if sut.refresh.saved[0].UserID() != sut.user.ID() {
		t.Fatalf("saved userID mismatch")
	}
}

func TestLoginUser_InvalidCredentials_BadEmailFormat(t *testing.T) {
	t.Parallel()

	sut := newLoginSUTFull(t)
	_, err := sut.uc.Execute(context.Background(), usecase.LoginUserInput{
		Email:    "x",
		Password: "12345678",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginUser_InvalidCredentials_UserNotFound(t *testing.T) {
	t.Parallel()

	sut := newLoginSUTFull(t)
	_, err := sut.uc.Execute(context.Background(), usecase.LoginUserInput{
		Email:    "missing@example.com",
		Password: sut.password,
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginUser_InvalidCredentials_WrongPassword(t *testing.T) {
	t.Parallel()

	sut := newLoginSUTFull(t)
	_, err := sut.uc.Execute(context.Background(), usecase.LoginUserInput{
		Email:    sut.user.Email().String(),
		Password: "wrong-password",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginUser_DummyHashCalledOnUserNotFound(t *testing.T) {
	t.Parallel()

	sut := newLoginSUTFull(t)
	if sut.hasher.verifyCalls != 0 {
		t.Fatalf("verifyCalls != 0 at start")
	}
	_, _ = sut.uc.Execute(context.Background(), usecase.LoginUserInput{
		Email:    "missing@example.com",
		Password: sut.password,
	})
	if sut.hasher.verifyCalls != 1 {
		t.Fatalf("verifyCalls = %d, want 1 (timing alignment)", sut.hasher.verifyCalls)
	}
}

func TestLoginUser_RandReadFails(t *testing.T) {
	t.Parallel()

	sut := newLoginSUTFull(t)
	sut.rand.err = errors.New("rand boom")

	_, err := sut.uc.Execute(context.Background(), usecase.LoginUserInput{
		Email:    sut.user.Email().String(),
		Password: sut.password,
	})
	if err == nil || errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected wrapped rand error, got %v", err)
	}
	if len(sut.refresh.saved) != 0 {
		t.Fatalf("refresh repo should be empty")
	}
	if sut.hasher.verifyCalls != 1 {
		t.Fatalf("hasher.Verify must be called before rand: verifyCalls = %d", sut.hasher.verifyCalls)
	}
}
