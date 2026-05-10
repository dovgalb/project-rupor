package httpauth

import (
	"github.com/go-chi/chi/v5"

	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
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
		r.Group(func(r chi.Router) {
			r.Post("/register", NewRegisterHandler(deps.Register).ServeHTTP)
			r.Post("/login", NewLoginHandler(deps.Login).ServeHTTP)
			r.Post("/refresh", NewRefreshHandler(deps.Refresh).ServeHTTP)
		})
		r.Group(func(r chi.Router) {
			r.Use(authmw.RequireAuth(deps.TokenIssuer, deps.Clock))
			r.Get("/me", NewMeHandler(deps.Me).ServeHTTP)
		})
	})
}
