package websocket

import (
	"fmt"

	"github.com/google/uuid"
)

type TopicKind uint8

const (
	TopicKindUnknown TopicKind = iota
	TopicKindChannel
	TopicKindRoom
)

type Topic struct {
	kind TopicKind
	id   uuid.UUID
}

func ChannelTopic(channelID uuid.UUID) Topic { return Topic{kind: TopicKindChannel, id: channelID} }
func RoomTopic(roomID uuid.UUID) Topic       { return Topic{kind: TopicKindRoom, id: roomID} }

func (t Topic) Kind() TopicKind { return t.kind }
func (t Topic) ID() uuid.UUID   { return t.id }

func (t Topic) Key() string {
	switch t.kind {
	case TopicKindChannel:
		return "channel:" + t.id.String()
	case TopicKindRoom:
		return "room:" + t.id.String()
	default:
		return fmt.Sprintf("unknown:%s", t.id)
	}
}
