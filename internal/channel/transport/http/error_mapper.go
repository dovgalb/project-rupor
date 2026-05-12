package httpchannel

import (
	"errors"
	"net/http"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/channel/domain"
	"github.com/dovgalb/project-rupor/pkg/httpx"
)

type httpError struct {
	status int
	code   string
	msg    string
}

func mapError(err error) httpError {
	switch {
	// 400.
	case errors.Is(err, domain.ErrInvalidChannelName):
		return httpError{http.StatusBadRequest, "CHANNEL-001", "invalid channel name"}
	case errors.Is(err, domain.ErrInvalidChannelKind):
		return httpError{http.StatusBadRequest, "CHANNEL-002", "invalid channel kind"}

	// 404.
	case errors.Is(err, domain.ErrChannelNotFound):
		return httpError{http.StatusNotFound, "CHANNEL-003", "channel or room not found"}

	// 409.
	case errors.Is(err, domain.ErrChannelNameAlreadyTaken):
		return httpError{http.StatusConflict, "CHANNEL-004", "channel name already taken"}

	// 403.
	case errors.Is(err, domain.ErrChannelAccessDenied):
		return httpError{http.StatusForbidden, "CHANNEL-006", "access denied: not a member"}
	case errors.Is(err, domain.ErrChannelInsufficientRole):
		return httpError{http.StatusForbidden, "CHANNEL-007", "insufficient role: admin or owner required"}

	// 401 — защитная ветка при !ok в UserIDFromContext.
	case errors.Is(err, authdom.ErrAccessTokenInvalid):
		return httpError{http.StatusUnauthorized, "AUTH-010", "access token invalid"}

	default:
		return httpError{http.StatusInternalServerError, "INTERNAL", "internal"}
	}
}

func writeError(w http.ResponseWriter, e httpError) {
	httpx.WriteJSONError(w, e.status, e.code, e.msg)
}

func writeBadBody(w http.ResponseWriter) {
	writeError(w, httpError{http.StatusBadRequest, "CHANNEL-005", "invalid request body"})
}
