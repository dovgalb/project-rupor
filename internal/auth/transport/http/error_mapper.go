package httpauth

import (
	"errors"
	"net/http"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/pkg/httpx"
)

type httpError struct {
	status  int
	code    string
	message string
}

func mapError(err error) httpError {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail):
		return httpError{http.StatusBadRequest, "AUTH-001", "invalid email"}
	case errors.Is(err, domain.ErrInvalidUsername):
		return httpError{http.StatusBadRequest, "AUTH-002", "invalid username"}
	case errors.Is(err, domain.ErrInvalidPassword):
		return httpError{http.StatusBadRequest, "AUTH-003", "invalid password"}
	case errors.Is(err, domain.ErrEmailAlreadyTaken):
		return httpError{http.StatusConflict, "AUTH-004", "email already taken"}
	case errors.Is(err, domain.ErrUsernameAlreadyTaken):
		return httpError{http.StatusConflict, "AUTH-005", "username already taken"}
	case errors.Is(err, domain.ErrInvalidCredentials):
		return httpError{http.StatusUnauthorized, "AUTH-006", "invalid credentials"}
	case errors.Is(err, domain.ErrRefreshTokenNotFound):
		return httpError{http.StatusUnauthorized, "AUTH-007", "refresh token not found"}
	case errors.Is(err, domain.ErrRefreshTokenRevoked):
		return httpError{http.StatusUnauthorized, "AUTH-008", "refresh token revoked"}
	case errors.Is(err, domain.ErrRefreshTokenExpired):
		return httpError{http.StatusUnauthorized, "AUTH-009", "refresh token expired"}
	case errors.Is(err, domain.ErrAccessTokenInvalid),
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrInvalidUserID):
		return httpError{http.StatusUnauthorized, "AUTH-010", "access token invalid"}
	case errors.Is(err, domain.ErrAccessTokenExpired):
		return httpError{http.StatusUnauthorized, "AUTH-011", "access token expired"}
	default:
		return httpError{http.StatusInternalServerError, "INTERNAL", "internal"}
	}
}

func writeError(w http.ResponseWriter, he httpError) {
	httpx.WriteJSONError(w, he.status, he.code, he.message)
}

func writeBadBody(w http.ResponseWriter) {
	httpx.WriteJSONError(w, http.StatusBadRequest, "AUTH-012", "malformed request body")
}
