package domain

const (
	minPasswordLength = 8
	maxPasswordLength = 72
)

type Password struct{ value string }

func NewPassword(raw string) (Password, error) {
	l := len(raw)
	if l < minPasswordLength || l > maxPasswordLength {
		return Password{}, ErrInvalidPassword
	}
	return Password{value: raw}, nil
}

// String возвращает raw-пароль для передачи в bcrypt-адаптер.
// Не использовать для логирования и сериализации.
func (p Password) String() string { return p.value }
