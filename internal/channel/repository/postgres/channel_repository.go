package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
	"github.com/dovgalb/project-rupor/internal/channel/repository/postgres/db"
)

const channelsRoomIDNameKey = "channels_room_id_name_key"

type ChannelRepository struct {
	q *db.Queries
}

func NewChannelRepository(pool *pgxpool.Pool) *ChannelRepository {
	return &ChannelRepository{q: db.New(pool)}
}

func (r *ChannelRepository) Save(ctx context.Context, ch *domain.Channel) error {
	if err := r.q.InsertChannel(ctx, domainToInsertChannelParams(ch)); err != nil {
		if isUniqueViolation(err, channelsRoomIDNameKey) {
			return domain.ErrChannelNameAlreadyTaken
		}
		return fmt.Errorf("postgres: insert channel: %w", err)
	}
	return nil
}

func (r *ChannelRepository) ListByRoom(
	ctx context.Context,
	roomID domain.RoomID,
) ([]*domain.Channel, error) {
	rows, err := r.q.ListChannelsByRoom(ctx, roomID.UUID())
	if err != nil {
		return nil, fmt.Errorf("postgres: list channels by room: %w", err)
	}
	out := make([]*domain.Channel, 0, len(rows))
	for _, row := range rows {
		ch, mErr := channelRowToDomain(row)
		if mErr != nil {
			return nil, mErr
		}
		out = append(out, ch)
	}
	return out, nil
}

func (r *ChannelRepository) DeleteInRoom(
	ctx context.Context,
	channelID domain.ChannelID,
	roomID domain.RoomID,
) error {
	affected, err := r.q.DeleteChannelInRoom(ctx, db.DeleteChannelInRoomParams{
		ID:     channelID.UUID(),
		RoomID: roomID.UUID(),
	})
	if err != nil {
		return fmt.Errorf("postgres: delete channel: %w", err)
	}
	if affected == 0 {
		return domain.ErrChannelNotFound
	}
	return nil
}
