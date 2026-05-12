package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
	"github.com/dovgalb/project-rupor/internal/chat/repository/postgres/db"
	"github.com/dovgalb/project-rupor/internal/chat/usecase"
)

type MessageRepository struct {
	q *db.Queries
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{q: db.New(pool)}
}

func (r *MessageRepository) Save(ctx context.Context, m *domain.Message) error {
	if err := r.q.InsertMessage(ctx, domainToInsertMessageParams(m)); err != nil {
		return fmt.Errorf("postgres: insert message: %w", err)
	}
	return nil
}

func (r *MessageRepository) ListByChannel(
	ctx context.Context,
	channelID domain.ChannelID,
	before domain.MessageID,
	limit int,
) ([]*domain.Message, error) {
	rows, err := r.q.ListMessagesBeforeCursor(ctx, db.ListMessagesBeforeCursorParams{
		ChannelID: channelID.UUID(),
		BeforeID:  optionalUUID(before),
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("postgres: list messages: %w", err)
	}
	out := make([]*domain.Message, 0, len(rows))
	for _, row := range rows {
		m, mErr := messageRowToDomain(row)
		if mErr != nil {
			return nil, mErr
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *MessageRepository) ChannelOf(
	ctx context.Context,
	channelID domain.ChannelID,
) (usecase.ChannelInfo, error) {
	row, err := r.q.GetChannelKind(ctx, channelID.UUID())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return usecase.ChannelInfo{}, domain.ErrChannelNotFound
		}
		return usecase.ChannelInfo{}, fmt.Errorf("postgres: get channel kind: %w", err)
	}
	roomID, err := domain.NewRoomID(row.RoomID)
	if err != nil {
		return usecase.ChannelInfo{}, fmt.Errorf("postgres: channel row: room_id: %w", err)
	}
	return usecase.ChannelInfo{
		ChannelID: channelID,
		RoomID:    roomID,
		Kind:      row.Kind,
	}, nil
}
