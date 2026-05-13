package httpchat

import (
	"encoding/json"
	"io"
	"time"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
)

// messageResponse — DTO одного сообщения в HTTP-ответе.
type messageResponse struct {
	ID        string    `json:"id"`
	ChannelID string    `json:"channelId"`
	AuthorID  string    `json:"authorId"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

// listMessagesResponse — DTO страницы сообщений с курсором следующей страницы.
type listMessagesResponse struct {
	Items      []messageResponse `json:"items"`
	NextBefore *string           `json:"nextBefore"`
}

// messageToResponse маппит доменное сообщение в DTO ответа.
func messageToResponse(m *domain.Message) messageResponse {
	return messageResponse{
		ID:        m.ID().String(),
		ChannelID: m.ChannelID().String(),
		AuthorID:  m.AuthorID().String(),
		Text:      m.Text().String(),
		CreatedAt: m.CreatedAt(),
	}
}

// jsonEncode сериализует значение в JSON и пишет в w.
func jsonEncode(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
