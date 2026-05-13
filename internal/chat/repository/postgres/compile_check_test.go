package postgres_test

import (
	pg "github.com/dovgalb/project-rupor/internal/chat/repository/postgres"
	chuc "github.com/dovgalb/project-rupor/internal/chat/usecase"
)

var _ chuc.MessageRepository = (*pg.MessageRepository)(nil)
