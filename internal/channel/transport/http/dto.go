package httpchannel

import (
	"encoding/json"
	"io"
	"time"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

type createChannelRequest struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type channelResponse struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"roomId"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"createdAt"`
}

type listChannelsResponse struct {
	Items []channelResponse `json:"items"`
}

func channelToResponse(c *domain.Channel) channelResponse {
	return channelResponse{
		ID:        c.ID().String(),
		RoomID:    c.RoomID().String(),
		Name:      c.Name().String(),
		Kind:      c.Kind().String(),
		CreatedAt: c.CreatedAt(),
	}
}

func jsonDecode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func jsonEncode(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
