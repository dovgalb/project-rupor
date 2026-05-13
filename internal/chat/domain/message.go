package domain

import "time"

// Message — доменная сущность сообщения в текстовом канале.
type Message struct {
	id        MessageID
	channelID ChannelID
	authorID  UserID
	text      MessageText
	createdAt time.Time
}

// NewMessage собирает сообщение и проверяет инварианты идентификаторов и createdAt.
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

// ReconstructMessage восстанавливает сообщение из репозитория, применяя те же инварианты, что и NewMessage.
func ReconstructMessage(
	id MessageID,
	channelID ChannelID,
	authorID UserID,
	text MessageText,
	createdAt time.Time,
) (*Message, error) {
	return NewMessage(id, channelID, authorID, text, createdAt)
}

// ID возвращает идентификатор сообщения.
func (m *Message) ID() MessageID { return m.id }

// ChannelID возвращает идентификатор канала, в который отправлено сообщение.
func (m *Message) ChannelID() ChannelID { return m.channelID }

// AuthorID возвращает идентификатор автора сообщения.
func (m *Message) AuthorID() UserID { return m.authorID }

// Text возвращает текст сообщения.
func (m *Message) Text() MessageText { return m.text }

// CreatedAt возвращает момент создания сообщения в UTC.
func (m *Message) CreatedAt() time.Time { return m.createdAt }
