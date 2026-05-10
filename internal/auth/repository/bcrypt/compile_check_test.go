package bcrypt_test

import (
	"github.com/dovgalb/project-rupor/internal/auth/repository/bcrypt"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
)

var _ usecase.PasswordHasher = (*bcrypt.PasswordHasher)(nil)
