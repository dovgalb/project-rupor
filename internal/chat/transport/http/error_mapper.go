package httpchat

import (
	"errors"
	"net/http"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/chat/domain"
	"github.com/dovgalb/project-rupor/pkg/httpx"
)

type httpError struct {
	status int
	code   string
	msg    string
}

func mapError(err error) httpError {
	switch {
	case errors.Is(err, domain.ErrInvalidMessageText):
		return httpError{http.StatusBadRequest, "CHAT-001", "invalid message text"}
	case errors.Is(err, domain.ErrChannelNotFound):
		return httpError{http.StatusNotFound, "CHAT-002", "channel not found"}
	case errors.Is(err, domain.ErrChannelNotText):
		return httpError{http.StatusBadRequest, "CHAT-003", "channel is not text"}
	case errors.Is(err, domain.ErrChatAccessDenied),
		errors.Is(err, domain.ErrChatInsufficientRole):
		return httpError{http.StatusForbidden, "CHAT-004", "access denied: not a room member"}
	case errors.Is(err, domain.ErrInvalidMessageID),
		errors.Is(err, domain.ErrInvalidChannelID),
		errors.Is(err, domain.ErrInvalidAuthorID),
		errors.Is(err, domain.ErrInvalidRoomID):
		return httpError{http.StatusBadRequest, "CHAT-005", "invalid uuid in path/query"}
	case errors.Is(err, authdom.ErrAccessTokenInvalid):
		return httpError{http.StatusUnauthorized, "AUTH-010", "access token invalid"}
	default:
		return httpError{http.StatusInternalServerError, "INTERNAL", "internal"}
	}
}

func writeError(w http.ResponseWriter, e httpError) {
	httpx.WriteJSONError(w, e.status, e.code, e.msg)
}

func writeLimitError(w http.ResponseWriter) {
	writeError(w, httpError{http.StatusBadRequest, "CHAT-006", "limit must be 1..100"})
}

func writeBadUUIDError(w http.ResponseWriter) {
	writeError(w, httpError{http.StatusBadRequest, "CHAT-005", "invalid uuid in path/query"})
}
