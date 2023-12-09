package websocket

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/gorilla/websocket"
)

type BaseMessage struct {
	Type    string `json:"type"`
	Id      int64  `json:"id"`
	Success bool   `json:"success"`
}

type ChanMsg struct {
	Id      int64
	Type    string
	Success bool
	Raw     []byte
}

func ListenWebsocket(conn *websocket.Conn, ctx context.Context, c chan ChanMsg) {
	for {
		bytes, err := ReadMessage(conn, ctx)
		if err != nil {
			slog.Error("Error reading from websocket:", err)
			close(c)
			break
		}

		base := BaseMessage{
			// default to true for messages that don't include "success" at all
			Success: true,
		}
		json.Unmarshal(bytes, &base)
		if !base.Success {
			slog.Warn("Received unsuccessful response", "response", string(bytes))
		}
		chanMsg := ChanMsg{
			Type:    base.Type,
			Id:      base.Id,
			Success: base.Success,
			Raw:     bytes,
		}

		c <- chanMsg
	}
}
