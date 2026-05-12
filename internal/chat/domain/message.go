package domain

import "time"

type Message struct {
	id        MessageID
	channelID ChannelID
	authorID  UserID
	text      MessageText
	createdAt time.Time
}

func NewMessage(
	id MessageID,
	channelID ChannelID,
	authorID UserID,
	text MessageText,
	createdAt time.Time,
) (*Message, error) {
	if id.IsZero() {
		return nil, ErrInvalidMessageID
	}
	if channelID.IsZero() {
		return nil, ErrInvalidChannelID
	}
	if authorID.IsZero() {
		return nil, ErrInvalidAuthorID
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidCreatedAt
	}
	return &Message{
		id:        id,
		channelID: channelID,
		authorID:  authorID,
		text:      text,
		createdAt: createdAt,
	}, nil
}

func ReconstructMessage(
	id MessageID,
	channelID ChannelID,
	authorID UserID,
	text MessageText,
	createdAt time.Time,
) (*Message, error) {
	return NewMessage(id, channelID, authorID, text, createdAt)
}

func (m *Message) ID() MessageID        { return m.id }
func (m *Message) ChannelID() ChannelID { return m.channelID }
func (m *Message) AuthorID() UserID     { return m.authorID }
func (m *Message) Text() MessageText    { return m.text }
func (m *Message) CreatedAt() time.Time { return m.createdAt }
