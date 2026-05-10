package httpauth

import (
	"net/http"

	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

type RegisterHandler struct {
	uc *usecase.RegisterUser
}

func NewRegisterHandler(uc *usecase.RegisterUser) *RegisterHandler {
	return &RegisterHandler{uc: uc}
}

func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := jsonDecode(r.Body, &req); err != nil {
		writeBadBody(w)
		return
	}

	out, err := h.uc.Execute(r.Context(), usecase.RegisterUserInput{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = jsonEncode(w, registerResponse{
		ID:        out.UserID,
		Email:     out.Email,
		Username:  out.Username,
		CreatedAt: out.CreatedAt,
	})
}
