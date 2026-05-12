package httpchat

import (
	"github.com/go-chi/chi/v5"

	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	authuc "github.com/dovgalb/project-rupor/internal/auth/usecase"
	"github.com/dovgalb/project-rupor/internal/chat/usecase"
)

type Deps struct {
	ListMessages *usecase.ListMessages
	TokenIssuer  authuc.TokenIssuer
	Clock        authuc.Clock
}

func RegisterRoutes(r chi.Router, deps Deps) {
	r.Route("/channels/{channelID}/messages", func(r chi.Router) {
		r.Use(authmw.RequireAuth(deps.TokenIssuer, deps.Clock))
		r.Get("/", NewListMessagesHandler(deps.ListMessages).ServeHTTP)
	})
}
