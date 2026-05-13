package websocket

import (
	"net/http"

	ws "github.com/coder/websocket"
)

const defaultMaxFrameBytes = 64 * 1024

type UpgradeOptions struct {
	// OriginPatterns — список паттернов origin (host или *.host) для CORS-проверки upgrade.
	OriginPatterns []string
	// MaxFrameBytes — лимит размера входящего фрейма; 0 → 64 KB.
	MaxFrameBytes int64
}

func Upgrade(w http.ResponseWriter, r *http.Request, opts UpgradeOptions) (*ws.Conn, error) {
	conn, err := ws.Accept(w, r, &ws.AcceptOptions{
		OriginPatterns:     opts.OriginPatterns,
		InsecureSkipVerify: false,
	})
	if err != nil {
		return nil, err
	}
	limit := opts.MaxFrameBytes
	if limit == 0 {
		limit = defaultMaxFrameBytes
	}
	conn.SetReadLimit(limit)
	return conn, nil
}
