package middleware

import (
	"context"

	"github.com/dovgalb/project-rupor/internal/auth/domain"
)

type userIDKey struct{}

// WithUserID кладёт валидированный domain.UserID в контекст.
func WithUserID(ctx context.Context, id domain.UserID) context.Context {
	return context.WithValue(ctx, userIDKey{}, id)
}

// UserIDFromContext извлекает UserID из контекста. Второе значение — флаг присутствия.
func UserIDFromContext(ctx context.Context) (domain.UserID, bool) {
	v := ctx.Value(userIDKey{})
	if v == nil {
		return domain.UserID{}, false
	}
	id, ok := v.(domain.UserID)
	return id, ok
}
