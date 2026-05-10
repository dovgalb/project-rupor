package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

// fakeUserRepo — однопоточный фейк UserRepository.
type fakeUserRepo struct {
	byID    map[string]*domain.User
	byEmail map[string]*domain.User
	saved   []*domain.User
	saveErr error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:    map[string]*domain.User{},
		byEmail: map[string]*domain.User{},
	}
}

func (r *fakeUserRepo) Save(_ context.Context, u *domain.User) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, u)
	r.byID[u.ID().String()] = u
	r.byEmail[u.Email().String()] = u
	return nil
}

func (r *fakeUserRepo) FindByID(_ context.Context, id domain.UserID) (*domain.User, error) {
	u, ok := r.byID[id.String()]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, e domain.Email) (*domain.User, error) {
	u, ok := r.byEmail[e.String()]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

// fakeRefreshRepo — однопоточный фейк RefreshTokenRepository.
type fakeRefreshRepo struct {
	byHash    map[[32]byte]*domain.RefreshToken
	saved     []*domain.RefreshToken
	rotateOld []domain.TokenHash
	rotateNew []*domain.RefreshToken
	saveErr   error
	findErr   error
	rotateErr error
}

func newFakeRefreshRepo() *fakeRefreshRepo {
	return &fakeRefreshRepo{byHash: map[[32]byte]*domain.RefreshToken{}}
}

func (r *fakeRefreshRepo) Save(_ context.Context, t *domain.RefreshToken) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, t)
	r.byHash[t.TokenHash().Bytes()] = t
	return nil
}

func (r *fakeRefreshRepo) FindByHash(_ context.Context, h domain.TokenHash) (*domain.RefreshToken, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	t, ok := r.byHash[h.Bytes()]
	if !ok {
		return nil, domain.ErrRefreshTokenNotFound
	}
	return t, nil
}

func (r *fakeRefreshRepo) Rotate(_ context.Context, oldHash domain.TokenHash, newToken *domain.RefreshToken, _ time.Time) error {
	if r.rotateErr != nil {
		return r.rotateErr
	}
	r.rotateOld = append(r.rotateOld, oldHash)
	r.rotateNew = append(r.rotateNew, newToken)
	r.byHash[newToken.TokenHash().Bytes()] = newToken
	return nil
}

// fakeHasher — мапит Password → PasswordHash через лукап. verifyCalls — счётчик.
type fakeHasher struct {
	hashes       map[string]domain.PasswordHash
	verifyCalls  int
	hashErr      error
	verifyErr    error
	verifyReject map[string]struct{} // (hash + "|" + password) → mismatch
}

func newFakeHasher() *fakeHasher {
	return &fakeHasher{hashes: map[string]domain.PasswordHash{}, verifyReject: map[string]struct{}{}}
}

func (h *fakeHasher) Hash(p domain.Password) (domain.PasswordHash, error) {
	if h.hashErr != nil {
		return domain.PasswordHash{}, h.hashErr
	}
	stored, ok := h.hashes[p.String()]
	if ok {
		return stored, nil
	}
	hashed, err := domain.NewPasswordHash("hashed:" + p.String())
	if err != nil {
		return domain.PasswordHash{}, err
	}
	return hashed, nil
}

func (h *fakeHasher) Verify(stored domain.PasswordHash, p domain.Password) error {
	h.verifyCalls++
	if h.verifyErr != nil {
		return h.verifyErr
	}
	key := stored.String() + "|" + p.String()
	if _, reject := h.verifyReject[key]; reject {
		return domain.ErrInvalidCredentials
	}
	expected, ok := h.hashes[p.String()]
	if ok && expected == stored {
		return nil
	}
	if stored.String() == "hashed:"+p.String() {
		return nil
	}
	return domain.ErrInvalidCredentials
}

// fakeIssuer — детерминированный TokenIssuer.
type fakeIssuer struct {
	issued    []domain.UserID
	accessTTL time.Duration
	issueErr  error
}

func (i *fakeIssuer) IssueAccess(userID domain.UserID, now time.Time) (string, time.Time, error) {
	if i.issueErr != nil {
		return "", time.Time{}, i.issueErr
	}
	i.issued = append(i.issued, userID)
	return "access-" + userID.String(), now.Add(i.accessTTL), nil
}

func (i *fakeIssuer) VerifyAccess(_ string, _ time.Time) (domain.UserID, error) {
	return domain.UserID{}, errors.New("not used in tests")
}

// fixedClock — стабильное время.
type fixedClock struct{ now time.Time }

func (c *fixedClock) Now() time.Time { return c.now }

// fixedUUID — выдаёт заранее заданные UUID по очереди.
type fixedUUID struct {
	next []uuid.UUID
	idx  int
}

func (g *fixedUUID) New() uuid.UUID {
	id := g.next[g.idx]
	g.idx++
	return id
}

// fixedRand — заранее заданные байтовые буферы.
type fixedRand struct {
	next [][]byte
	idx  int
	err  error
}

func (r *fixedRand) Read(_ int) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	b := r.next[r.idx]
	r.idx++
	return b, nil
}

// helpers.

func mustEmail(t *testing.T, raw string) domain.Email {
	t.Helper()
	v, err := domain.NewEmail(raw)
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	return v
}

func mustPasswordHash(t *testing.T, raw string) domain.PasswordHash {
	t.Helper()
	v, err := domain.NewPasswordHash(raw)
	if err != nil {
		t.Fatalf("NewPasswordHash: %v", err)
	}
	return v
}

func mustUsername(t *testing.T, raw string) domain.Username {
	t.Helper()
	v, err := domain.NewUsername(raw)
	if err != nil {
		t.Fatalf("NewUsername: %v", err)
	}
	return v
}

func mustUserID(t *testing.T, raw uuid.UUID) domain.UserID {
	t.Helper()
	v, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("NewUserID: %v", err)
	}
	return v
}

func mustRefreshTokenID(t *testing.T, raw uuid.UUID) domain.RefreshTokenID {
	t.Helper()
	v, err := domain.NewRefreshTokenID(raw)
	if err != nil {
		t.Fatalf("NewRefreshTokenID: %v", err)
	}
	return v
}

func mustTokenHash(t *testing.T, raw []byte) domain.TokenHash {
	t.Helper()
	v, err := domain.NewTokenHash(raw)
	if err != nil {
		t.Fatalf("NewTokenHash: %v", err)
	}
	return v
}

func mustUser(t *testing.T, email, username, hash string, createdAt time.Time) *domain.User {
	t.Helper()
	u, err := domain.NewUser(
		mustUserID(t, uuid.New()),
		mustEmail(t, email),
		mustUsername(t, username),
		mustPasswordHash(t, hash),
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	return u
}
