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

type DeleteRoomHandler struct {
	uc *usecase.DeleteRoom
}

func NewDeleteRoomHandler(uc *usecase.DeleteRoom) *DeleteRoomHandler {
	return &DeleteRoomHandler{uc: uc}
}

func (h *DeleteRoomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid, ok := authmw.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, mapError(authdom.ErrAccessTokenInvalid))
		return
	}

	roomID, err := uuid.Parse(chi.URLParam(r, "roomID"))
	if err != nil {
		writeError(w, mapError(domain.ErrRoomNotFound))
		return
	}

	if execErr := h.uc.Execute(r.Context(), usecase.DeleteRoomInput{
		ActorID: uid.UUID(),
		RoomID:  roomID,
	}); execErr != nil {
		writeError(w, mapDeleteRoomError(execErr))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
