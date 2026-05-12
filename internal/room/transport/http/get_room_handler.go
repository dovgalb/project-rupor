package httproom

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

type GetRoomHandler struct {
	uc *usecase.GetRoom
}

func NewGetRoomHandler(uc *usecase.GetRoom) *GetRoomHandler {
	return &GetRoomHandler{uc: uc}
}

func (h *GetRoomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid, ok := authmw.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, mapError(authdom.ErrAccessTokenInvalid))
		return
	}

	roomID, err := uuid.Parse(chi.URLParam(r, "roomID"))
	if err != nil {
		// Невалидный UUID → ресурс не существует.
		writeError(w, mapError(domain.ErrRoomNotFound))
		return
	}

	out, err := h.uc.Execute(r.Context(), usecase.GetRoomInput{
		ActorID: uid.UUID(),
		RoomID:  roomID,
	})
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = jsonEncode(w, roomToResponse(out.Room))
}
