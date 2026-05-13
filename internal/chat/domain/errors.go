package domain

import "errors"

// Валидация VO/Entity.
var (
	ErrInvalidMessageID   = errors.New("chat: invalid message id")
	ErrInvalidChannelID   = errors.New("chat: invalid channel id")
	ErrInvalidAuthorID    = errors.New("chat: invalid author id")
	ErrInvalidRoomID      = errors.New("chat: invalid room id")
	ErrInvalidMessageText = errors.New("chat: invalid message text")
	ErrInvalidCreatedAt   = errors.New("chat: created_at must not be zero")
)

// Бизнес-инварианты.
var (
	ErrMessageNotFound      = errors.New("chat: message not found")
	ErrChannelNotFound      = errors.New("chat: channel not found")
	ErrChannelNotText       = errors.New("chat: channel is not text")
	ErrChatAccessDenied     = errors.New("chat: access denied")
	ErrChatInsufficientRole = errors.New("chat: insufficient role for the operation")
)
