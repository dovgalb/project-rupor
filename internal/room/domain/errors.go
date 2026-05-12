package domain

import "errors"

// Валидация VO/Entity.
var (
	ErrInvalidRoomID     = errors.New("room: invalid room id")
	ErrInvalidUserID     = errors.New("room: invalid user id")
	ErrInvalidInviteID   = errors.New("room: invalid invite id")
	ErrInvalidCreatedAt  = errors.New("room: created_at must not be zero")
	ErrInvalidRoomName   = errors.New("room: invalid room name")
	ErrInvalidRole       = errors.New("room: invalid role")
	ErrInvalidInviteCode = errors.New("room: invalid invite code")
)

// Бизнес-инварианты.
var (
	ErrRoomNotFound         = errors.New("room: room not found")
	ErrNotMember            = errors.New("room: not a member of the room")
	ErrAlreadyMember        = errors.New("room: already a member of the room")
	ErrInsufficientRole     = errors.New("room: insufficient role for the operation")
	ErrInviteNotFound       = errors.New("room: invite not found")
	ErrInviteAlreadyRevoked = errors.New("room: invite already revoked")
	ErrCannotDemoteOwner    = errors.New("room: cannot demote owner")
)
