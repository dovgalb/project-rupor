package httproom

import (
	"net/http"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

type CreateRoomHandler struct {
	uc *usecase.CreateRoom
}

func NewCreateRoomHandler(uc *usecase.CreateRoom) *CreateRoomHandler {
	return &CreateRoomHandler{uc: uc}
}

func (h *CreateRoomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid, ok := authmw.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, mapError(authdom.ErrAccessTokenInvalid))
		return
	}

	var req createRoomRequest
	if err := jsonDecode(r.Body, &req); err != nil {
		writeBadBody(w)
		return
	}

	out, err := h.uc.Execute(r.Context(), usecase.CreateRoomInput{
		ActorID: uid.UUID(),
		Name:    req.Name,
	})
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = jsonEncode(w, roomToResponse(out.Room))
}
