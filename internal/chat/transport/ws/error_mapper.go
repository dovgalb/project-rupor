package wschat

import (
	"errors"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
)

// mapDomainError транслирует доменную/usecase-ошибку в код+сообщение для
// WS-error-фрейма (без HTTP-статуса — это отдельный канал доставки).
func mapDomainError(err error) (code, msg string) {
	switch {
	case errors.Is(err, domain.ErrInvalidMessageText):
		return "CHAT-001", "invalid message text"
	case errors.Is(err, domain.ErrChannelNotFound):
		return "CHAT-002", "channel not found"
	case errors.Is(err, domain.ErrChannelNotText):
		return "CHAT-003", "channel is not text"
	case errors.Is(err, domain.ErrChatAccessDenied),
		errors.Is(err, domain.ErrChatInsufficientRole):
		return "CHAT-004", "access denied: not a room member"
	case errors.Is(err, domain.ErrInvalidMessageID),
		errors.Is(err, domain.ErrInvalidChannelID),
		errors.Is(err, domain.ErrInvalidAuthorID),
		errors.Is(err, domain.ErrInvalidRoomID):
		return "CHAT-005", "invalid uuid in payload"
	default:
		return "INTERNAL", "internal"
	}
}
