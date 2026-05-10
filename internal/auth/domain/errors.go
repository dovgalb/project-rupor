package domain

import "errors"

// Валидация VO/Entity.
var (
	ErrInvalidEmail                  = errors.New("auth: invalid email")
	ErrInvalidUsername               = errors.New("auth: invalid username")
	ErrInvalidPassword               = errors.New("auth: invalid password")
	ErrInvalidPasswordHash           = errors.New("auth: invalid password hash")
	ErrInvalidUserID                 = errors.New("auth: invalid user id")
	ErrInvalidRefreshTokenID         = errors.New("auth: invalid refresh token id")
	ErrInvalidTokenHash              = errors.New("auth: invalid token hash")
	ErrInvalidRefreshTokenExpiration = errors.New("auth: refresh token expiration must be after creation")
	ErrInvalidCreatedAt              = errors.New("auth: created_at must not be zero")
)

// Бизнес-правила User/RefreshToken.
var (
	ErrEmailAlreadyTaken          = errors.New("auth: email already taken")
	ErrUsernameAlreadyTaken       = errors.New("auth: username already taken")
	ErrUserNotFound               = errors.New("auth: user not found")
	ErrInvalidCredentials         = errors.New("auth: invalid credentials")
	ErrRefreshTokenNotFound       = errors.New("auth: refresh token not found")
	ErrRefreshTokenRevoked        = errors.New("auth: refresh token revoked")
	ErrRefreshTokenExpired        = errors.New("auth: refresh token expired")
	ErrRefreshTokenAlreadyRevoked = errors.New("auth: refresh token already revoked")
	ErrAccessTokenInvalid         = errors.New("auth: access token invalid")
	ErrAccessTokenExpired         = errors.New("auth: access token expired")
)
