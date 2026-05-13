package domain

import "github.com/google/uuid"

// RoomID — идентификатор комнаты, value object с гарантией ненулевого UUID.
type RoomID struct{ value uuid.UUID }

// NewRoomID создаёт RoomID, отклоняя uuid.Nil.
func NewRoomID(raw uuid.UUID) (RoomID, error) {
	if raw == uuid.Nil {
		return RoomID{}, ErrInvalidRoomID
	}
	return RoomID{value: raw}, nil
}

// UUID возвращает обёрнутое значение uuid.UUID.
func (id RoomID) UUID() uuid.UUID { return id.value }

// String возвращает каноническое строковое представление UUID.
func (id RoomID) String() string { return id.value.String() }

// IsZero сообщает, что идентификатор не инициализирован.
func (id RoomID) IsZero() bool { return id.value == uuid.Nil }
