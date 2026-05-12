package wschat

import (
	"github.com/go-chi/chi/v5"

	authuc "github.com/dovgalb/project-rupor/internal/auth/usecase"
	"github.com/dovgalb/project-rupor/internal/chat/usecase"
	pws "github.com/dovgalb/project-rupor/pkg/websocket"
)

type WSDeps struct {
	Hub                *pws.Hub
	SendMessage        *usecase.SendMessage
	MembershipForChat  usecase.MembershipQuery
	MembershipForRooms MembershipReader
	TokenIssuer        authuc.TokenIssuer
	Clock              authuc.Clock
	OriginPatterns     []string
}

func RegisterWSRoute(r chi.Router, deps WSDeps) {
	r.Get("/ws", NewWSHandler(deps).ServeHTTP)
}
