package domain

import "github.com/google/uuid"

type RefreshTokenID struct{ value uuid.UUID }

func NewRefreshTokenID(raw uuid.UUID) (RefreshTokenID, error) {
	if raw == uuid.Nil {
		return RefreshTokenID{}, ErrInvalidRefreshTokenID
	}
	return RefreshTokenID{value: raw}, nil
}

func (id RefreshTokenID) UUID() uuid.UUID { return id.value }
func (id RefreshTokenID) String() string  { return id.value.String() }
func (id RefreshTokenID) IsZero() bool    { return id.value == uuid.Nil }
