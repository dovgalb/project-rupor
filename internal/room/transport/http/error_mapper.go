package httproom

import (
	"errors"
	"net/http"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/room/domain"
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
	case errors.Is(err, domain.ErrInvalidRoomName):
		return httpError{http.StatusBadRequest, "ROOM-001", "invalid room name"}
	case errors.Is(err, domain.ErrInvalidInviteCode):
		return httpError{http.StatusBadRequest, "ROOM-008", "invalid invite code"}

	// 404.
	case errors.Is(err, domain.ErrRoomNotFound):
		return httpError{http.StatusNotFound, "ROOM-002", "room not found"}
	case errors.Is(err, domain.ErrInviteNotFound):
		return httpError{http.StatusNotFound, "ROOM-007", "invite not found or revoked"}

	// 403.
	case errors.Is(err, domain.ErrNotMember):
		return httpError{http.StatusForbidden, "ROOM-003", "not a member"}
	case errors.Is(err, domain.ErrInsufficientRole):
		return httpError{http.StatusForbidden, "ROOM-004", "insufficient role: admin or owner required"}

	// 409.
	case errors.Is(err, domain.ErrAlreadyMember):
		return httpError{http.StatusConflict, "ROOM-006", "already a member"}

	// 401 — защитная ветка при !ok в UserIDFromContext (после RequireAuth не должно).
	case errors.Is(err, authdom.ErrAccessTokenInvalid):
		return httpError{http.StatusUnauthorized, "AUTH-010", "access token invalid"}

	default:
		return httpError{http.StatusInternalServerError, "INTERNAL", "internal"}
	}
}

// mapDeleteRoomError — обёртка для UC-R4: ErrInsufficientRole здесь означает
// «требуется owner», возвращается ROOM-005 вместо ROOM-004.
func mapDeleteRoomError(err error) httpError {
	if errors.Is(err, domain.ErrInsufficientRole) {
		return httpError{http.StatusForbidden, "ROOM-005", "only owner can delete room"}
	}
	return mapError(err)
}

func writeError(w http.ResponseWriter, e httpError) {
	httpx.WriteJSONError(w, e.status, e.code, e.msg)
}

func writeBadBody(w http.ResponseWriter) {
	writeError(w, httpError{http.StatusBadRequest, "ROOM-009", "invalid request body"})
}
