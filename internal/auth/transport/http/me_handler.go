package httpauth

import (
	"net/http"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

type MeHandler struct {
	uc *usecase.GetCurrentUser
}

func NewMeHandler(uc *usecase.GetCurrentUser) *MeHandler {
	return &MeHandler{uc: uc}
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid, ok := authmw.UserIDFromContext(r.Context())
	if !ok {
		// Защитная ветка: handler оказался без RequireAuth по ошибке маршрутизации.
		writeError(w, mapError(domain.ErrAccessTokenInvalid))
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
