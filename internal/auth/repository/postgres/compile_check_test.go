package postgres_test

import (
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

var (
	_ usecase.UserRepository         = (*postgres.UserRepository)(nil)
	_ usecase.RefreshTokenRepository = (*postgres.RefreshTokenRepository)(nil)
)
