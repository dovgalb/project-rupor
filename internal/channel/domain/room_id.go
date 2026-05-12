package domain

import "github.com/google/uuid"

// RoomID — собственный value object идентификатора комнаты в channel-домене.
// Параллелен room.domain.RoomID; домены не импортируют друг друга (см. 03-decisions.md §D-08).
// Мост через uuid.UUID реализован в MembershipQueryAdapter.
type RoomID struct{ value uuid.UUID }

func NewRoomID(raw uuid.UUID) (RoomID, error) {
	if raw == uuid.Nil {
		return RoomID{}, ErrInvalidRoomID
	}
	return RoomID{value: raw}, nil
}

func (id RoomID) UUID() uuid.UUID { return id.value }
func (id RoomID) String() string  { return id.value.String() }
func (id RoomID) IsZero() bool    { return id.value == uuid.Nil }
