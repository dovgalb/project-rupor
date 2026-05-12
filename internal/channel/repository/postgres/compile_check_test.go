package postgres_test

import (
	pg "github.com/dovgalb/project-rupor/internal/channel/repository/postgres"
	chuc "github.com/dovgalb/project-rupor/internal/channel/usecase"
)

var _ chuc.ChannelRepository = (*pg.ChannelRepository)(nil)
