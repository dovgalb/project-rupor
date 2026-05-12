package postgres

import (
	"fmt"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
	"github.com/dovgalb/project-rupor/internal/channel/repository/postgres/db"
)

func channelRowToDomain(row db.Channel) (*domain.Channel, error) {
	id, err := domain.NewChannelID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("channel row: id: %w", err)
	}
	roomID, err := domain.NewRoomID(row.RoomID)
	if err != nil {
		return nil, fmt.Errorf("channel row: room_id: %w", err)
	}
	name, err := domain.NewChannelName(row.Name)
	if err != nil {
		return nil, fmt.Errorf("channel row: name: %w", err)
	}
	kind, err := domain.ParseChannelKind(row.Kind)
	if err != nil {
		return nil, fmt.Errorf("channel row: kind: %w", err)
	}
	return domain.ReconstructChannel(id, roomID, name, kind, row.CreatedAt)
}

func domainToInsertChannelParams(c *domain.Channel) db.InsertChannelParams {
	return db.InsertChannelParams{
		ID:        c.ID().UUID(),
		RoomID:    c.RoomID().UUID(),
		Name:      c.Name().String(),
		Kind:      c.Kind().String(),
		CreatedAt: c.CreatedAt(),
	}
}
