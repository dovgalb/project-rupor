package httpx

import "context"

type requestIDKey struct{}

// WithRequestID кладёт request_id в контекст. Валидацию значения обеспечивает middleware-фабрика.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestIDFromContext извлекает request_id из контекста.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIDKey{}).(string)
	return v, ok
}
