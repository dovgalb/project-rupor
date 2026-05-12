package domain

import "github.com/google/uuid"

type MessageID struct{ value uuid.UUID }

func NewMessageID(raw uuid.UUID) (MessageID, error) {
	if raw == uuid.Nil {
		return MessageID{}, ErrInvalidMessageID
	}
	return MessageID{value: raw}, nil
}

func (id MessageID) UUID() uuid.UUID { return id.value }
func (id MessageID) String() string  { return id.value.String() }
func (id MessageID) IsZero() bool    { return id.value == uuid.Nil }
