package middleware

import "net/http"

// responseWriter — общая обёртка, отслеживающая статус и факт WriteHeader.
// Используется Recover и Logger.
type responseWriter struct {
	http.ResponseWriter
	status        int
	headerWritten bool
}

// wrap возвращает обёртку. Идемпотентен: повторное оборачивание возвращает тот же экземпляр.
func wrap(w http.ResponseWriter) *responseWriter {
	if rw, ok := w.(*responseWriter); ok {
		return rw
	}
	return &responseWriter{ResponseWriter: w}
}

func (r *responseWriter) WriteHeader(code int) {
	if r.headerWritten {
		return
	}
	r.status = code
	r.headerWritten = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseWriter) Write(b []byte) (int, error) {
	if !r.headerWritten {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}
