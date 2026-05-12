package domain

import "time"

type Channel struct {
	id        ChannelID
	roomID    RoomID
	name      ChannelName
	kind      ChannelKind
	createdAt time.Time
}

func NewChannel(
	id ChannelID,
	roomID RoomID,
	name ChannelName,
	kind ChannelKind,
	createdAt time.Time,
) (*Channel, error) {
	if id.IsZero() {
		return nil, ErrInvalidChannelID
	}
	if roomID.IsZero() {
		return nil, ErrInvalidRoomID
	}
	if !kind.IsValid() {
		return nil, ErrInvalidChannelKind
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidCreatedAt
	}
	return &Channel{
		id:        id,
		roomID:    roomID,
		name:      name,
		kind:      kind,
		createdAt: createdAt,
	}, nil
}

func ReconstructChannel(
	id ChannelID,
	roomID RoomID,
	name ChannelName,
	kind ChannelKind,
	createdAt time.Time,
) (*Channel, error) {
	return NewChannel(id, roomID, name, kind, createdAt)
}

func (c *Channel) ID() ChannelID        { return c.id }
func (c *Channel) RoomID() RoomID       { return c.roomID }
func (c *Channel) Name() ChannelName    { return c.name }
func (c *Channel) Kind() ChannelKind    { return c.kind }
func (c *Channel) CreatedAt() time.Time { return c.createdAt }
