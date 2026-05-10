package bcrypt

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

type PasswordHasher struct {
	cost int
}

func NewPasswordHasher(cost int) *PasswordHasher {
	return &PasswordHasher{cost: cost}
}

func (h *PasswordHasher) Hash(p domain.Password) (domain.PasswordHash, error) {
	raw, err := bcrypt.GenerateFromPassword([]byte(p.String()), h.cost)
	if err != nil {
		return domain.PasswordHash{}, fmt.Errorf("bcrypt: hash: %w", err)
	}
	return domain.NewPasswordHash(string(raw))
}

func (h *PasswordHasher) Verify(stored domain.PasswordHash, p domain.Password) error {
	err := bcrypt.CompareHashAndPassword([]byte(stored.String()), []byte(p.String()))
	if err == nil {
		return nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return domain.ErrInvalidCredentials
	}
	return fmt.Errorf("bcrypt: verify: %w", err)
}
