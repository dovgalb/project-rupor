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

type inboundEvent struct {
	Type      string `json:"type"`
	ChannelID string `json:"channel_id,omitempty"`
	Text      string `json:"text,omitempty"`
}

func errorFrame(code, msg string) map[string]any {
	return map[string]any{
		"type": OutError,
		"data": map[string]any{
			"code":    code,
			"message": msg,
		},
	}
}

func subscribedFrame(channelID uuid.UUID) map[string]any {
	return map[string]any{
		"type": OutSubscribed,
		"data": map[string]any{"channel_id": channelID.String()},
	}
}

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
