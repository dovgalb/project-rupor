package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
	"github.com/dovgalb/project-rupor/pkg/httpx"
)

const bearerPrefix = "Bearer "

// RequireAuth — middleware, валидирующий Authorization: Bearer <jwt>.
// При успехе кладёт domain.UserID в контекст, иначе отвечает 401 + AUTH-010/011.
func RequireAuth(issuer usecase.TokenIssuer, clock usecase.Clock) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, bearerPrefix) || len(auth) <= len(bearerPrefix) {
				httpx.WriteJSONError(w, http.StatusUnauthorized, "AUTH-010", "access token invalid")
				return
			}
			token := auth[len(bearerPrefix):]

			uid, err := issuer.VerifyAccess(token, clock.Now())
			if err != nil {
				status, code, msg := mapAuthError(err)
				httpx.WriteJSONError(w, status, code, msg)
				return
			}

			ctx := WithUserID(r.Context(), uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func mapAuthError(err error) (int, string, string) {
	if errors.Is(err, domain.ErrAccessTokenExpired) {
		return http.StatusUnauthorized, "AUTH-011", "access token expired"
	}
	return http.StatusUnauthorized, "AUTH-010", "access token invalid"
}
