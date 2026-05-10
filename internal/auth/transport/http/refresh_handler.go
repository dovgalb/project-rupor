package httpauth

import (
	"net/http"

	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

type RefreshHandler struct {
	uc *usecase.RefreshAccess
}

func NewRefreshHandler(uc *usecase.RefreshAccess) *RefreshHandler {
	return &RefreshHandler{uc: uc}
}

func (h *RefreshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := jsonDecode(r.Body, &req); err != nil {
		writeBadBody(w)
		return
	}

	out, err := h.uc.Execute(r.Context(), usecase.RefreshAccessInput{
		RefreshToken: req.RefreshToken,
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
