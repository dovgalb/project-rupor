package httpauth_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	xbcrypt "golang.org/x/crypto/bcrypt"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/repository/bcrypt"
	"github.com/dovgalb/project-rupor/internal/auth/repository/jwt"
	httpauth "github.com/dovgalb/project-rupor/internal/auth/transport/http"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

const (
	testAccessTTL  = 15 * time.Minute
	testRefreshTTL = 24 * time.Hour
)

var testJWTSecret = []byte("test-secret-1234567890")

// fakeUserRepo, fakeRefreshRepo — копии из usecase-тестов (другой пакет, тесты их не видят).

type fakeUserRepo struct {
	mu      sync.Mutex
	byID    map[string]*domain.User
	byEmail map[string]*domain.User
	saveErr error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byID: map[string]*domain.User{}, byEmail: map[string]*domain.User{}}
}

func (r *fakeUserRepo) Save(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveErr != nil {
		return r.saveErr
	}
	if _, dup := r.byEmail[u.Email().String()]; dup {
		return domain.ErrEmailAlreadyTaken
	}
	r.byID[u.ID().String()] = u
	r.byEmail[u.Email().String()] = u
	return nil
}

func (r *fakeUserRepo) FindByID(_ context.Context, id domain.UserID) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[id.String()]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, e domain.Email) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byEmail[e.String()]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

type fakeRefreshRepo struct {
	mu        sync.Mutex
	byHash    map[[32]byte]*domain.RefreshToken
	rotateErr error
	saveErr   error
}

func newFakeRefreshRepo() *fakeRefreshRepo {
	return &fakeRefreshRepo{byHash: map[[32]byte]*domain.RefreshToken{}}
}

func (r *fakeRefreshRepo) Save(_ context.Context, t *domain.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveErr != nil {
		return r.saveErr
	}
	r.byHash[t.TokenHash().Bytes()] = t
	return nil
}

func (r *fakeRefreshRepo) FindByHash(_ context.Context, h domain.TokenHash) (*domain.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byHash[h.Bytes()]
	if !ok {
		return nil, domain.ErrRefreshTokenNotFound
	}
	return t, nil
}

func (r *fakeRefreshRepo) Rotate(_ context.Context, oldHash domain.TokenHash, newToken *domain.RefreshToken, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rotateErr != nil {
		return r.rotateErr
	}
	old, ok := r.byHash[oldHash.Bytes()]
	if !ok {
		return domain.ErrRefreshTokenNotFound
	}
	if err := old.Revoke(revokedAt); err != nil {
		if errors.Is(err, domain.ErrRefreshTokenAlreadyRevoked) {
			return domain.ErrRefreshTokenRevoked
		}
		return err
	}
	r.byHash[newToken.TokenHash().Bytes()] = newToken
	return nil
}

type realClock struct{ now time.Time }

func (c *realClock) Now() time.Time { return c.now }

type seqUUID struct {
	mu  sync.Mutex
	idx int
}

func (g *seqUUID) New() uuid.UUID {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.idx++
	return uuid.New()
}

type seqRand struct {
	mu  sync.Mutex
	idx int
	err error
}

func (r *seqRand) Read(n int) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	r.idx++
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(r.idx)
	}
	return out, nil
}

type testEnv struct {
	server  *httptest.Server
	users   *fakeUserRepo
	refresh *fakeRefreshRepo
	clock   *realClock
	rand    *seqRand
	hasher  *bcrypt.PasswordHasher
	issuer  *jwt.TokenIssuer
}

func setupServer(t *testing.T) *testEnv {
	t.Helper()

	users := newFakeUserRepo()
	refresh := newFakeRefreshRepo()
	hasher := bcrypt.NewPasswordHasher(xbcrypt.MinCost)
	issuer := jwt.NewTokenIssuer(testJWTSecret, testAccessTTL)

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	clock := &realClock{now: now}
	rand := &seqRand{}

	dummyPwd, err := domain.NewPassword("dummy12345")
	if err != nil {
		t.Fatalf("NewPassword(dummy): %v", err)
	}
	dummyHash, err := hasher.Hash(dummyPwd)
	if err != nil {
		t.Fatalf("Hash(dummy): %v", err)
	}

	register := usecase.NewRegisterUser(users, hasher, clock, &seqUUID{})
	login := usecase.NewLoginUser(users, refresh, hasher, issuer, clock, &seqUUID{}, rand, testRefreshTTL, dummyHash)
	refreshUC := usecase.NewRefreshAccess(refresh, issuer, clock, &seqUUID{}, rand, testRefreshTTL)
	me := usecase.NewGetCurrentUser(users)

	r := chi.NewRouter()
	httpauth.RegisterRoutes(r, httpauth.Deps{
		Register:    register,
		Login:       login,
		Refresh:     refreshUC,
		Me:          me,
		TokenIssuer: issuer,
		Clock:       clock,
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &testEnv{
		server: srv, users: users, refresh: refresh,
		clock: clock, rand: rand, hasher: hasher, issuer: issuer,
	}
}
