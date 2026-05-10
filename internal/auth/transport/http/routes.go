package httpauth

import (
	"github.com/go-chi/chi/v5"

	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

type Deps struct {
	Register    *usecase.RegisterUser
	Login       *usecase.LoginUser
	Refresh     *usecase.RefreshAccess
	Me          *usecase.GetCurrentUser
	TokenIssuer usecase.TokenIssuer
	Clock       usecase.Clock
}

func RegisterRoutes(r chi.Router, deps Deps) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", NewRegisterHandler(deps.Register).ServeHTTP)
		r.Post("/login", NewLoginHandler(deps.Login).ServeHTTP)
		r.Post("/refresh", NewRefreshHandler(deps.Refresh).ServeHTTP)
		r.Get("/me", NewMeHandler(deps.Me, deps.TokenIssuer, deps.Clock).ServeHTTP)
	})
}
