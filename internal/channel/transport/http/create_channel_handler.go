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

type CreateChannelHandler struct {
	uc *usecase.CreateChannel
}

func NewCreateChannelHandler(uc *usecase.CreateChannel) *CreateChannelHandler {
	return &CreateChannelHandler{uc: uc}
}

func (h *CreateChannelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var req createChannelRequest
	if dErr := jsonDecode(r.Body, &req); dErr != nil {
		writeBadBody(w)
		return
	}

	out, err := h.uc.Execute(r.Context(), usecase.CreateChannelInput{
		ActorID: uid.UUID(),
		RoomID:  roomID,
		Name:    req.Name,
		Kind:    req.Kind,
	})
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = jsonEncode(w, channelToResponse(out.Channel))
}
