package httpchannel

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	"github.com/dovgalb/project-rupor/internal/channel/domain"
	"github.com/dovgalb/project-rupor/internal/channel/usecase"
)

type DeleteChannelHandler struct {
	uc *usecase.DeleteChannel
}

func NewDeleteChannelHandler(uc *usecase.DeleteChannel) *DeleteChannelHandler {
	return &DeleteChannelHandler{uc: uc}
}

func (h *DeleteChannelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid, ok := authmw.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, mapError(authdom.ErrAccessTokenInvalid))
		return
	}

	roomID, err := uuid.Parse(chi.URLParam(r, "roomID"))
	if err != nil {
		writeError(w, mapError(domain.ErrChannelNotFound))
		return
	}
	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		writeError(w, mapError(domain.ErrChannelNotFound))
		return
	}

	if execErr := h.uc.Execute(r.Context(), usecase.DeleteChannelInput{
		ActorID:   uid.UUID(),
		RoomID:    roomID,
		ChannelID: channelID,
	}); execErr != nil {
		writeError(w, mapError(execErr))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
