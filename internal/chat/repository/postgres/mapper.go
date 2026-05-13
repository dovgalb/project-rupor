package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
	"github.com/dovgalb/project-rupor/internal/chat/repository/postgres/db"
)

// messageRowToDomain восстанавливает доменное сообщение из строки таблицы messages.
func messageRowToDomain(row db.Message) (*domain.Message, error) {
	id, err := domain.NewMessageID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("message row: id: %w", err)
	}
	channelID, err := domain.NewChannelID(row.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("message row: channel_id: %w", err)
	}
	authorID, err := domain.NewUserID(row.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("message row: author_id: %w", err)
	}
	text, err := domain.NewMessageText(row.Text)
	if err != nil {
		return nil, fmt.Errorf("message row: text: %w", err)
	}
	return domain.ReconstructMessage(id, channelID, authorID, text, row.CreatedAt.UTC())
}

// domainToInsertMessageParams готовит параметры sqlc-запроса InsertMessage из доменного сообщения.
func domainToInsertMessageParams(m *domain.Message) db.InsertMessageParams {
	return db.InsertMessageParams{
		ID:        m.ID().UUID(),
		ChannelID: m.ChannelID().UUID(),
		AuthorID:  m.AuthorID().UUID(),
		Text:      m.Text().String(),
		CreatedAt: m.CreatedAt().UTC(),
	}
}

// optionalUUID превращает MessageID в pgtype.UUID; для zero-value MessageID
// возвращает невалидное (NULL) значение.
func optionalUUID(id domain.MessageID) pgtype.UUID {
	if id.IsZero() {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: id.UUID(), Valid: true}
}
