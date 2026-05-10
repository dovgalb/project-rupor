package httpauth

import (
	"net/http"

	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

type LoginHandler struct {
	uc *usecase.LoginUser
}

func NewLoginHandler(uc *usecase.LoginUser) *LoginHandler {
	return &LoginHandler{uc: uc}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := jsonDecode(r.Body, &req); err != nil {
		writeBadBody(w)
		return
	}

	out, err := h.uc.Execute(r.Context(), usecase.LoginUserInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = jsonEncode(w, tokensResponse{
		AccessToken:      out.AccessToken,
		RefreshToken:     out.RefreshToken,
		AccessExpiresAt:  out.AccessExpiresAt,
		RefreshExpiresAt: out.RefreshExpiresAt,
	})
}
