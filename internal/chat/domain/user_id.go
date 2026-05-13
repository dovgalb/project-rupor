package domain

import "github.com/google/uuid"

// UserID — идентификатор пользователя в контексте чата, value object с гарантией ненулевого UUID.
type UserID struct{ value uuid.UUID }

// NewUserID создаёт UserID, отклоняя uuid.Nil.
func NewUserID(raw uuid.UUID) (UserID, error) {
	if raw == uuid.Nil {
		return UserID{}, ErrInvalidAuthorID
	}
	return UserID{value: raw}, nil
}

// UUID возвращает обёрнутое значение uuid.UUID.
func (id UserID) UUID() uuid.UUID { return id.value }

// String возвращает каноническое строковое представление UUID.
func (id UserID) String() string { return id.value.String() }

// IsZero сообщает, что идентификатор не инициализирован.
func (id UserID) IsZero() bool { return id.value == uuid.Nil }
