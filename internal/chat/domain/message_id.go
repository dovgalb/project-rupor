package domain

import "github.com/google/uuid"

// MessageID — идентификатор сообщения, value object с гарантией ненулевого UUID.
type MessageID struct{ value uuid.UUID }

// NewMessageID создаёт MessageID, отклоняя uuid.Nil.
func NewMessageID(raw uuid.UUID) (MessageID, error) {
	if raw == uuid.Nil {
		return MessageID{}, ErrInvalidMessageID
	}
	return MessageID{value: raw}, nil
}

// UUID возвращает обёрнутое значение uuid.UUID.
func (id MessageID) UUID() uuid.UUID { return id.value }

// String возвращает каноническое строковое представление UUID.
func (id MessageID) String() string { return id.value.String() }

// IsZero сообщает, что идентификатор не инициализирован.
func (id MessageID) IsZero() bool { return id.value == uuid.Nil }
