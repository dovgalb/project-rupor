package main

import (
	"context"
	"net/url"
	"time"

	"github.com/google/uuid"

	chatdom "github.com/dovgalb/project-rupor/internal/chat/domain"
	roomdom "github.com/dovgalb/project-rupor/internal/room/domain"
	roompg "github.com/dovgalb/project-rupor/internal/room/repository/postgres"
	pws "github.com/dovgalb/project-rupor/pkg/websocket"
)

// hubChatBroadcaster адаптирует *pws.Hub под chat/usecase.Broadcaster.
type hubChatBroadcaster struct{ hub *pws.Hub }

func newHubChatBroadcaster(hub *pws.Hub) *hubChatBroadcaster {
	return &hubChatBroadcaster{hub: hub}
}

func (b *hubChatBroadcaster) PublishToChannel(channelID chatdom.ChannelID, eventType string, payload any) {
	b.hub.Publish(context.Background(), pws.ChannelTopic(channelID.UUID()), eventType, payload)
}

func (b *hubChatBroadcaster) PublishToRoom(roomID chatdom.RoomID, eventType string, payload any) {
	b.hub.Publish(context.Background(), pws.RoomTopic(roomID.UUID()), eventType, payload)
}

// hubRoomEventsPublisher адаптирует *pws.Hub под room/usecase.RoomEventsPublisher.
type hubRoomEventsPublisher struct{ hub *pws.Hub }

func newHubRoomEventsPublisher(hub *pws.Hub) *hubRoomEventsPublisher {
	return &hubRoomEventsPublisher{hub: hub}
}

func (p *hubRoomEventsPublisher) PublishMemberJoined(roomID roomdom.RoomID, userID roomdom.UserID, joinedAt time.Time) {
	payload := map[string]any{
		"room_id":   roomID.String(),
		"user_id":   userID.String(),
		"joined_at": joinedAt.Format(time.RFC3339Nano),
	}
	p.hub.Publish(context.Background(), pws.RoomTopic(roomID.UUID()), "member.joined", payload)
}

// roomIDsAdapter адаптирует *roompg.RoomRepository.ListByMember под
// wschat.MembershipReader (нужен только список UUID комнат пользователя).
type roomIDsAdapter struct{ repo *roompg.RoomRepository }

func (a roomIDsAdapter) ListRoomIDsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	uid, err := roomdom.NewUserID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := a.repo.ListByMember(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Room.ID().UUID())
	}
	return out, nil
}

// stripScheme приводит CORS origin'ы к формату OriginPatterns coder/websocket
// (host без схемы). Пустые/невалидные origin'ы отбрасываются.
func stripScheme(origins []string) []string {
	out := make([]string, 0, len(origins))
	for _, o := range origins {
		u, err := url.Parse(o)
		if err != nil || u.Host == "" {
			continue
		}
		out = append(out, u.Host)
	}
	return out
}
