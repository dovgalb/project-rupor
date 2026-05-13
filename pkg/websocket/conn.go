package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	ws "github.com/coder/websocket"
	"github.com/google/uuid"
)

const writeTimeout = 5 * time.Second

type ConnID = uuid.UUID

type Conn struct {
	id        ConnID
	userID    uuid.UUID
	ws        *ws.Conn
	writeMu   sync.Mutex
	closeOnce sync.Once
}

func NewConn(userID uuid.UUID, raw *ws.Conn) *Conn {
	return &Conn{id: uuid.New(), userID: userID, ws: raw}
}

func (c *Conn) ID() ConnID        { return c.id }
func (c *Conn) UserID() uuid.UUID { return c.userID }

// WriteJSON сериализует payload и отправляет фрейм под mutex'ом.
func (c *Conn) WriteJSON(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.writeRaw(ctx, data)
}

// ReadJSON читает один JSON-фрейм. Вызывается только из owner-горутины.
func (c *Conn) ReadJSON(ctx context.Context, v any) error {
	_, data, err := c.ws.Read(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Close идемпотентен.
func (c *Conn) Close(code ws.StatusCode, reason string) {
	c.closeOnce.Do(func() {
		_ = c.ws.Close(code, reason)
	})
}

// writeRaw используется Hub'ом для broadcast'а уже сериализованного фрейма.
func (c *Conn) writeRaw(ctx context.Context, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return c.ws.Write(ctx, ws.MessageText, data)
}
