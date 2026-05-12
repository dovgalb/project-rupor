package httproom

import (
	"github.com/go-chi/chi/v5"

	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	authuc "github.com/dovgalb/project-rupor/internal/auth/usecase"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

type Deps struct {
	CreateRoom       *usecase.CreateRoom
	GetRoom          *usecase.GetRoom
	ListUserRooms    *usecase.ListUserRooms
	DeleteRoom       *usecase.DeleteRoom
	ListMembers      *usecase.ListMembers
	RegenerateInvite *usecase.RegenerateInvite
	JoinByCode       *usecase.JoinByCode
	TokenIssuer      authuc.TokenIssuer
	Clock            authuc.Clock
}

func RegisterRoutes(r chi.Router, deps Deps) {
	r.Route("/rooms", func(r chi.Router) {
		r.Use(authmw.RequireAuth(deps.TokenIssuer, deps.Clock))

		r.Post("/", NewCreateRoomHandler(deps.CreateRoom).ServeHTTP)
		r.Get("/", NewListRoomsHandler(deps.ListUserRooms).ServeHTTP)
		r.Get("/{roomID}", NewGetRoomHandler(deps.GetRoom).ServeHTTP)
		r.Delete("/{roomID}", NewDeleteRoomHandler(deps.DeleteRoom).ServeHTTP)
		r.Get("/{roomID}/members", NewListMembersHandler(deps.ListMembers).ServeHTTP)
		r.Post("/{roomID}/invite", NewRegenerateInviteHandler(deps.RegenerateInvite).ServeHTTP)
		r.Post("/join/{code}", NewJoinByCodeHandler(deps.JoinByCode).ServeHTTP)
	})
}
