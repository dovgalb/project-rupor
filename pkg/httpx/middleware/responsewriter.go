package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

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

// Hijack — нужен для WebSocket: библиотека coder/websocket делает type-assert
// http.Hijacker, чтобы перехватить TCP-соединение. Без этого upgrade падает с 501.
func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("middleware: ResponseWriter does not support Hijack")
	}
	// После Hijack контроль над соединением передан, статус-логика обёртки больше не работает.
	// Подставляем 101 чтобы access-лог отразил switching protocols.
	r.status = http.StatusSwitchingProtocols
	r.headerWritten = true
	return hj.Hijack()
}

// Flush — пробрасывает Flush к нижележащему writer'у (нужно для SSE/streaming).
func (r *responseWriter) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
