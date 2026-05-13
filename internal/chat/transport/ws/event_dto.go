package wschat

import (
	"github.com/google/uuid"
)

// Входящие типы.
const (
	EventTypeSubscribe   = "subscribe"
	EventTypeMessageSend = "message.send"
)

// Исходящие типы.
const (
	OutSubscribed  = "subscribed"
	OutMessageSent = "message.sent"
	OutError       = "error"
)

// inboundEvent — общий DTO входящего WS-фрейма (поля заполняются в зависимости от Type).
type inboundEvent struct {
	Type      string `json:"type"`
	ChannelID string `json:"channel_id,omitempty"`
	Text      string `json:"text,omitempty"`
}

// errorFrame собирает исходящий фрейм ошибки с кодом и сообщением.
func errorFrame(code, msg string) map[string]any {
	return map[string]any{
		"type": OutError,
		"data": map[string]any{
			"code":    code,
			"message": msg,
		},
	}
}

// subscribedFrame собирает фрейм подтверждения подписки на канал.
func subscribedFrame(channelID uuid.UUID) map[string]any {
	return map[string]any{
		"type": OutSubscribed,
		"data": map[string]any{"channel_id": channelID.String()},
	}
}

// messageSentFrame собирает фрейм-подтверждение успешной отправки сообщения автору запроса.
func messageSentFrame(messageID, channelID uuid.UUID, createdAt string) map[string]any {
	return map[string]any{
		"type": OutMessageSent,
		"data": map[string]any{
			"id":         messageID.String(),
			"channel_id": channelID.String(),
			"created_at": createdAt,
		},
	}
}
