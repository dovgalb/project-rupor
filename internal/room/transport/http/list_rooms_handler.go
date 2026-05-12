package httproom

import (
	"net/http"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

type ListRoomsHandler struct {
	uc *usecase.ListUserRooms
}

func NewListRoomsHandler(uc *usecase.ListUserRooms) *ListRoomsHandler {
	return &ListRoomsHandler{uc: uc}
}

func (h *ListRoomsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid, ok := authmw.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, mapError(authdom.ErrAccessTokenInvalid))
		return
	}

	out, err := h.uc.Execute(r.Context(), usecase.ListUserRoomsInput{
		ActorID: uid.UUID(),
	})
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	items := make([]roomWithRoleResponse, 0, len(out.Items))
	for _, rwr := range out.Items {
		items = append(items, roomWithRoleToResponse(rwr))
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = jsonEncode(w, listRoomsResponse{Items: items})
}
