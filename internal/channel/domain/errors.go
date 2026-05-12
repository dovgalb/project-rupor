package domain

import "errors"

// Валидация VO/Entity.
var (
	ErrInvalidChannelID   = errors.New("channel: invalid channel id")
	ErrInvalidRoomID      = errors.New("channel: invalid room id")
	ErrInvalidUserID      = errors.New("channel: invalid user id")
	ErrInvalidChannelName = errors.New("channel: invalid channel name")
	ErrInvalidChannelKind = errors.New("channel: invalid channel kind")
	ErrInvalidCreatedAt   = errors.New("channel: created_at must not be zero")
)

// Бизнес-инварианты.
var (
	ErrChannelNotFound         = errors.New("channel: channel not found")
	ErrChannelNameAlreadyTaken = errors.New("channel: channel name already taken in room")
	// ErrChannelAccessDenied — у пользователя нет membership в комнате.
	// Адаптер MembershipQueryAdapter транслирует room.domain.ErrNotMember сюда.
	ErrChannelAccessDenied = errors.New("channel: access denied")
	// ErrChannelInsufficientRole — у пользователя есть membership, но роли недостаточно.
	ErrChannelInsufficientRole = errors.New("channel: insufficient role for the operation")
)
