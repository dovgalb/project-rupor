package domain

import "time"

type Invite struct {
	id        InviteID
	roomID    RoomID
	code      InviteCode
	createdBy UserID
	createdAt time.Time
	// revokedAt zero == активен.
	revokedAt time.Time
}

func NewInvite(
	id InviteID,
	roomID RoomID,
	code InviteCode,
	createdBy UserID,
	createdAt time.Time,
) (*Invite, error) {
	if id.IsZero() {
		return nil, ErrInvalidInviteID
	}
	if roomID.IsZero() {
		return nil, ErrInvalidRoomID
	}
	if createdBy.IsZero() {
		return nil, ErrInvalidUserID
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidCreatedAt
	}
	return &Invite{
		id:        id,
		roomID:    roomID,
		code:      code,
		createdBy: createdBy,
		createdAt: createdAt,
		revokedAt: time.Time{},
	}, nil
}

// ReconstructInvite — восстановление инвайта из БД, с возможным заданным revokedAt
// (zero == активен).
func ReconstructInvite(
	id InviteID,
	roomID RoomID,
	code InviteCode,
	createdBy UserID,
	createdAt time.Time,
	revokedAt time.Time,
) (*Invite, error) {
	inv, err := NewInvite(id, roomID, code, createdBy, createdAt)
	if err != nil {
		return nil, err
	}
	inv.revokedAt = revokedAt
	return inv, nil
}

func (i *Invite) ID() InviteID         { return i.id }
func (i *Invite) RoomID() RoomID       { return i.roomID }
func (i *Invite) Code() InviteCode     { return i.code }
func (i *Invite) CreatedBy() UserID    { return i.createdBy }
func (i *Invite) CreatedAt() time.Time { return i.createdAt }
func (i *Invite) RevokedAt() time.Time { return i.revokedAt }

// Revoke помечает инвайт отозванным. Повторный Revoke возвращает ErrInviteAlreadyRevoked.
// zero now запрещён — защита от багов вызывающего (в реальном пути Clock.Now() не zero).
func (i *Invite) Revoke(now time.Time) error {
	if !i.revokedAt.IsZero() {
		return ErrInviteAlreadyRevoked
	}
	if now.IsZero() {
		return ErrInvalidCreatedAt
	}
	i.revokedAt = now
	return nil
}

func (i *Invite) IsActive() bool  { return i.revokedAt.IsZero() }
func (i *Invite) IsRevoked() bool { return !i.revokedAt.IsZero() }
