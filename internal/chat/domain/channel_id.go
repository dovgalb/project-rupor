package domain

import "github.com/google/uuid"

type ChannelID struct{ value uuid.UUID }

func NewChannelID(raw uuid.UUID) (ChannelID, error) {
	if raw == uuid.Nil {
		return ChannelID{}, ErrInvalidChannelID
	}
	return ChannelID{value: raw}, nil
}

func (id ChannelID) UUID() uuid.UUID { return id.value }
func (id ChannelID) String() string  { return id.value.String() }
func (id ChannelID) IsZero() bool    { return id.value == uuid.Nil }
