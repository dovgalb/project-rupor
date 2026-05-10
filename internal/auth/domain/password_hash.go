package domain

type PasswordHash struct{ value string }

func NewPasswordHash(raw string) (PasswordHash, error) {
	if raw == "" {
		return PasswordHash{}, ErrInvalidPasswordHash
	}
	return PasswordHash{value: raw}, nil
}

func (h PasswordHash) String() string { return h.value }
