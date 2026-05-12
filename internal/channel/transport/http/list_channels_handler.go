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

type ListChannelsHandler struct {
	uc *usecase.ListChannels
}

func NewListChannelsHandler(uc *usecase.ListChannels) *ListChannelsHandler {
	return &ListChannelsHandler{uc: uc}
}

func (h *ListChannelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.uc.Execute(r.Context(), usecase.ListChannelsInput{
		ActorID: uid.UUID(),
		RoomID:  roomID,
	})
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	items := make([]channelResponse, 0, len(out.Items))
	for _, ch := range out.Items {
		items = append(items, channelToResponse(ch))
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = jsonEncode(w, listChannelsResponse{Items: items})
}
