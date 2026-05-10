package jwt_test

import (
	"github.com/dovgalb/project-rupor/internal/auth/repository/jwt"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

var _ usecase.TokenIssuer = (*jwt.TokenIssuer)(nil)
