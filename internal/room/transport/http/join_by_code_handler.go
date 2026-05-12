package httproom

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

type JoinByCodeHandler struct {
	uc *usecase.JoinByCode
}

func NewJoinByCodeHandler(uc *usecase.JoinByCode) *JoinByCodeHandler {
	return &JoinByCodeHandler{uc: uc}
}

func (h *JoinByCodeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid, ok := authmw.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, mapError(authdom.ErrAccessTokenInvalid))
		return
	}

	code := chi.URLParam(r, "code")

	out, err := h.uc.Execute(r.Context(), usecase.JoinByCodeInput{
		ActorID: uid.UUID(),
		Code:    code,
	})
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = jsonEncode(w, roomToResponse(out.Room))
}
