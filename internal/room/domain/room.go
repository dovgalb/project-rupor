package domain

import "time"

type Room struct {
	id        RoomID
	ownerID   UserID
	name      RoomName
	createdAt time.Time
}

func NewRoom(id RoomID, ownerID UserID, name RoomName, createdAt time.Time) (*Room, error) {
	if id.IsZero() {
		return nil, ErrInvalidRoomID
	}
	if ownerID.IsZero() {
		return nil, ErrInvalidUserID
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidCreatedAt
	}
	return &Room{
		id:        id,
		ownerID:   ownerID,
		name:      name,
		createdAt: createdAt,
	}, nil
}

func ReconstructRoom(id RoomID, ownerID UserID, name RoomName, createdAt time.Time) (*Room, error) {
	return NewRoom(id, ownerID, name, createdAt)
}

func (r *Room) ID() RoomID           { return r.id }
func (r *Room) OwnerID() UserID      { return r.ownerID }
func (r *Room) Name() RoomName       { return r.name }
func (r *Room) CreatedAt() time.Time { return r.createdAt }

// TransferOwnership меняет владельца комнаты. Если newOwner zero — состояние не меняется.
// HTTP-эндпоинта в PR-2 нет, метод остаётся ради rich-domain (см. 03-decisions.md §D-17).
func (r *Room) TransferOwnership(newOwner UserID) error {
	if newOwner.IsZero() {
		return ErrInvalidUserID
	}
	r.ownerID = newOwner
	return nil
}
