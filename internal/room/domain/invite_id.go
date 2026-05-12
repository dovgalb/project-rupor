package domain

import "github.com/google/uuid"

type InviteID struct{ value uuid.UUID }

func NewInviteID(raw uuid.UUID) (InviteID, error) {
	if raw == uuid.Nil {
		return InviteID{}, ErrInvalidInviteID
	}
	return InviteID{value: raw}, nil
}

func (id InviteID) UUID() uuid.UUID { return id.value }
func (id InviteID) String() string  { return id.value.String() }
func (id InviteID) IsZero() bool    { return id.value == uuid.Nil }
