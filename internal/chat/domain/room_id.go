package domain

import "github.com/google/uuid"

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
