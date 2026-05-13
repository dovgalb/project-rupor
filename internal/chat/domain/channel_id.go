package domain

import "github.com/google/uuid"

// ChannelID — идентификатор канала, value object с гарантией ненулевого UUID.
type ChannelID struct{ value uuid.UUID }

// NewChannelID создаёт ChannelID, отклоняя uuid.Nil.
func NewChannelID(raw uuid.UUID) (ChannelID, error) {
	if raw == uuid.Nil {
		return ChannelID{}, ErrInvalidChannelID
	}
	return ChannelID{value: raw}, nil
}

// UUID возвращает обёрнутое значение uuid.UUID.
func (id ChannelID) UUID() uuid.UUID { return id.value }

// String возвращает каноническое строковое представление UUID.
func (id ChannelID) String() string { return id.value.String() }

// IsZero сообщает, что идентификатор не инициализирован.
func (id ChannelID) IsZero() bool { return id.value == uuid.Nil }
