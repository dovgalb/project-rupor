package httpauth

import (
	"net/http"
	"strings"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

const bearerPrefix = "Bearer "

type MeHandler struct {
	uc     *usecase.GetCurrentUser
	issuer usecase.TokenIssuer
	clock  usecase.Clock
}

func NewMeHandler(uc *usecase.GetCurrentUser, issuer usecase.TokenIssuer, clock usecase.Clock) *MeHandler {
	return &MeHandler{uc: uc, issuer: issuer, clock: clock}
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, bearerPrefix) || len(auth) <= len(bearerPrefix) {
		writeError(w, mapError(domain.ErrAccessTokenInvalid))
		return
	}
	token := auth[len(bearerPrefix):]

	uid, err := h.issuer.VerifyAccess(token, h.clock.Now())
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	out, err := h.uc.Execute(r.Context(), usecase.GetCurrentUserInput{UserID: uid.String()})
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = jsonEncode(w, meResponse{
		ID:        out.UserID,
		Email:     out.Email,
		Username:  out.Username,
		CreatedAt: out.CreatedAt,
	})
}
