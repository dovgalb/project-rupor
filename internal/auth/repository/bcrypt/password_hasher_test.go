package bcrypt_test

import (
	"errors"
	"testing"

	xbcrypt "golang.org/x/crypto/bcrypt"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/repository/bcrypt"
)

func mustPassword(t *testing.T, raw string) domain.Password {
	t.Helper()
	p, err := domain.NewPassword(raw)
	if err != nil {
		t.Fatalf("NewPassword: %v", err)
	}
	return p
}

func TestPasswordHasher_RoundTrip(t *testing.T) {
	t.Parallel()

	hasher := bcrypt.NewPasswordHasher(xbcrypt.MinCost)
	p := mustPassword(t, "correct-horse-battery")

	h, err := hasher.Hash(p)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if err := hasher.Verify(h, p); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestPasswordHasher_VerifyMismatch(t *testing.T) {
	t.Parallel()

	hasher := bcrypt.NewPasswordHasher(xbcrypt.MinCost)
	p := mustPassword(t, "correct-horse-battery")
	wrong := mustPassword(t, "wrong-pass-1234")

	h, err := hasher.Hash(p)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	err = hasher.Verify(h, wrong)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestPasswordHasher_VerifyInvalidHash(t *testing.T) {
	t.Parallel()

	hasher := bcrypt.NewPasswordHasher(xbcrypt.MinCost)
	stored, err := domain.NewPasswordHash("not-a-bcrypt-hash")
	if err != nil {
		t.Fatalf("NewPasswordHash: %v", err)
	}
	p := mustPassword(t, "correct-horse-battery")

	err = hasher.Verify(stored, p)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got ErrInvalidCredentials, want wrapped non-credential error: %v", err)
	}
}
