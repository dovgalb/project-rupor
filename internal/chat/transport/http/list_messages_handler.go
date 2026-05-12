package httpchat

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	"github.com/dovgalb/project-rupor/internal/chat/usecase"
)

type ListMessagesHandler struct {
	uc *usecase.ListMessages
}

func NewListMessagesHandler(uc *usecase.ListMessages) *ListMessagesHandler {
	return &ListMessagesHandler{uc: uc}
}

func (h *ListMessagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid, ok := authmw.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, mapError(authdom.ErrAccessTokenInvalid))
		return
	}

	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		writeBadUUIDError(w)
		return
	}

	var beforeID uuid.UUID
	if raw := r.URL.Query().Get("before"); raw != "" {
		beforeID, err = uuid.Parse(raw)
		if err != nil {
			writeBadUUIDError(w)
			return
		}
	}

	limit := usecase.DefaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, perr := strconv.Atoi(raw)
		if perr != nil || n < usecase.MinLimit || n > usecase.MaxLimit {
			writeLimitError(w)
			return
		}
		limit = n
	}

	out, err := h.uc.Execute(r.Context(), usecase.ListMessagesInput{
		ActorID:   uid.UUID(),
		ChannelID: channelID,
		Before:    beforeID,
		Limit:     limit,
	})
	if err != nil {
		writeError(w, mapError(err))
		return
	}

	items := make([]messageResponse, 0, len(out.Items))
	for _, m := range out.Items {
		items = append(items, messageToResponse(m))
	}
	var nextBeforeStr *string
	if out.NextBefore != uuid.Nil {
		s := out.NextBefore.String()
		nextBeforeStr = &s
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = jsonEncode(w, listMessagesResponse{Items: items, NextBefore: nextBeforeStr})
}
