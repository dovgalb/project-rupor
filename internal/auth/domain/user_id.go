package domain

import "github.com/google/uuid"

type UserID struct{ value uuid.UUID }

func NewUserID(raw uuid.UUID) (UserID, error) {
	if raw == uuid.Nil {
		return UserID{}, ErrInvalidUserID
	}
	return UserID{value: raw}, nil
}

func (id UserID) UUID() uuid.UUID { return id.value }
func (id UserID) String() string  { return id.value.String() }
func (id UserID) IsZero() bool    { return id.value == uuid.Nil }
