package httpchat

import (
	"encoding/json"
	"io"
	"time"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
)

type messageResponse struct {
	ID        string    `json:"id"`
	ChannelID string    `json:"channelId"`
	AuthorID  string    `json:"authorId"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type listMessagesResponse struct {
	Items      []messageResponse `json:"items"`
	NextBefore *string           `json:"nextBefore"`
}

func messageToResponse(m *domain.Message) messageResponse {
	return messageResponse{
		ID:        m.ID().String(),
		ChannelID: m.ChannelID().String(),
		AuthorID:  m.AuthorID().String(),
		Text:      m.Text().String(),
		CreatedAt: m.CreatedAt(),
	}
}

func jsonEncode(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
