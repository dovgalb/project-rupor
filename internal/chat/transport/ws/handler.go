package wschat

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	"github.com/dovgalb/project-rupor/internal/chat/domain"
	"github.com/dovgalb/project-rupor/internal/chat/usecase"
	"github.com/dovgalb/project-rupor/pkg/httpx"
	pws "github.com/dovgalb/project-rupor/pkg/websocket"
)

// WSHandler — HTTP-хендлер апгрейда соединения до WebSocket и обработки событий чата.
type WSHandler struct {
	deps WSDeps
}

// NewWSHandler собирает WSHandler из набора зависимостей.
func NewWSHandler(deps WSDeps) *WSHandler {
	return &WSHandler{deps: deps}
}

// ServeHTTP аутентифицирует клиента по query-параметру token, апгрейдит соединение и запускает readLoop.
// Подписка на room-topic'и пользователя выполняется автоматически при подключении.
func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := slog.Default()

	// Аутентификация ДО апгрейда — клиент должен суметь распарсить JSON-401.
	token := r.URL.Query().Get("token")
	if token == "" {
		httpx.WriteJSONError(w, http.StatusUnauthorized, "AUTH-010", "access token invalid")
		return
	}
	userID, err := h.deps.TokenIssuer.VerifyAccess(token, h.deps.Clock.Now())
	if err != nil {
		switch {
		case errors.Is(err, authdom.ErrAccessTokenExpired):
			httpx.WriteJSONError(w, http.StatusUnauthorized, "AUTH-011", "access token expired")
		default:
			httpx.WriteJSONError(w, http.StatusUnauthorized, "AUTH-010", "access token invalid")
		}
		return
	}

	// Список комнат пользователя — для авто-подписки.
	roomIDs, err := h.deps.MembershipForRooms.ListRoomIDsByUser(ctx, userID.UUID())
	if err != nil {
		logger.Error("ws: list room ids", slog.Any("err", err))
		httpx.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL", "internal")
		return
	}

	rawConn, err := pws.Upgrade(w, r, pws.UpgradeOptions{
		OriginPatterns: h.deps.OriginPatterns,
	})
	if err != nil {
		logger.Warn("ws: upgrade failed", slog.Any("err", err))
		return
	}

	conn := pws.NewConn(userID.UUID(), rawConn)
	cleanup := h.deps.Hub.Register(conn)
	defer cleanup()

	for _, roomID := range roomIDs {
		h.deps.Hub.Subscribe(conn, pws.RoomTopic(roomID))
	}

	h.readLoop(ctx, conn, userID.UUID())
}

// readLoop читает входящие фреймы и диспатчит их по типу события до закрытия соединения.
func (h *WSHandler) readLoop(ctx context.Context, conn *pws.Conn, userID uuid.UUID) {
	for {
		var in inboundEvent
		if err := conn.ReadJSON(ctx, &in); err != nil {
			return
		}
		switch in.Type {
		case EventTypeSubscribe:
			h.handleSubscribe(ctx, conn, userID, in)
		case EventTypeMessageSend:
			h.handleMessageSend(ctx, conn, userID, in)
		default:
			_ = conn.WriteJSON(ctx, errorFrame("CHAT-007", "unsupported event type"))
		}
	}
}

// handleSubscribe проверяет членство и подписывает соединение на topic канала.
func (h *WSHandler) handleSubscribe(ctx context.Context, conn *pws.Conn, userID uuid.UUID, in inboundEvent) {
	channelID, err := uuid.Parse(in.ChannelID)
	if err != nil {
		_ = conn.WriteJSON(ctx, errorFrame("CHAT-005", "invalid channel_id uuid"))
		return
	}
	chChannelID, err := domain.NewChannelID(channelID)
	if err != nil {
		_ = conn.WriteJSON(ctx, errorFrame("CHAT-005", "invalid channel_id"))
		return
	}
	chUserID, err := domain.NewUserID(userID)
	if err != nil {
		_ = conn.WriteJSON(ctx, errorFrame("CHAT-005", "invalid user_id"))
		return
	}
	if reqErr := h.deps.MembershipForChat.Require(ctx, chChannelID, chUserID, usecase.RoleAnyMember); reqErr != nil {
		code, msg := mapDomainError(reqErr)
		_ = conn.WriteJSON(ctx, errorFrame(code, msg))
		return
	}
	h.deps.Hub.Subscribe(conn, pws.ChannelTopic(channelID))
	_ = conn.WriteJSON(ctx, subscribedFrame(channelID))
}

// handleMessageSend делегирует сценарию SendMessage и отвечает автору фреймом message.sent.
func (h *WSHandler) handleMessageSend(ctx context.Context, conn *pws.Conn, userID uuid.UUID, in inboundEvent) {
	channelID, err := uuid.Parse(in.ChannelID)
	if err != nil {
		_ = conn.WriteJSON(ctx, errorFrame("CHAT-005", "invalid channel_id uuid"))
		return
	}
	out, err := h.deps.SendMessage.Execute(ctx, usecase.SendMessageInput{
		ActorID:   userID,
		ChannelID: channelID,
		Text:      in.Text,
	})
	if err != nil {
		code, msg := mapDomainError(err)
		_ = conn.WriteJSON(ctx, errorFrame(code, msg))
		return
	}
	_ = conn.WriteJSON(ctx, messageSentFrame(
		out.MessageID,
		out.ChannelID,
		out.CreatedAt.Format(time.RFC3339Nano),
	))
}
