package httpchannel

import (
	"github.com/go-chi/chi/v5"

	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	authuc "github.com/dovgalb/project-rupor/internal/auth/usecase"
	"github.com/dovgalb/project-rupor/internal/channel/usecase"
)

type Deps struct {
	CreateChannel *usecase.CreateChannel
	ListChannels  *usecase.ListChannels
	DeleteChannel *usecase.DeleteChannel
	TokenIssuer   authuc.TokenIssuer
	Clock         authuc.Clock
}

func RegisterRoutes(r chi.Router, deps Deps) {
	r.Route("/rooms/{roomID}/channels", func(r chi.Router) {
		r.Use(authmw.RequireAuth(deps.TokenIssuer, deps.Clock))

		r.Post("/", NewCreateChannelHandler(deps.CreateChannel).ServeHTTP)
		r.Get("/", NewListChannelsHandler(deps.ListChannels).ServeHTTP)
		r.Delete("/{channelID}", NewDeleteChannelHandler(deps.DeleteChannel).ServeHTTP)
	})
}
