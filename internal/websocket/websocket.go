// Package websocket is used to interact with the Home Assistant
// websocket API. All HA interaction is done via websocket
// except for cases explicitly called out in http package
// documentation.
package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	i "saml.dev/gome-assistant/internal"
)

var ErrInvalidToken = errors.New("invalid authentication token")

type AuthMessage struct {
	MsgType     string `json:"type"`
	AccessToken string `json:"access_token"`
}

type WebsocketWriter struct {
	Conn  *websocket.Conn
	mutex sync.Mutex
}

func (w *WebsocketWriter) WriteMessage(msg interface{}, ctx context.Context) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	err := w.Conn.WriteJSON(msg)
	if err != nil {
		return err
	}

	return nil
}

func ReadMessage(conn *websocket.Conn, ctx context.Context) ([]byte, error) {
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return []byte{}, err
	}
	return msg, nil
}

func SetupConnection(ip, port, authToken string) (*websocket.Conn, context.Context, context.CancelFunc, error) {
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second*3)

	// Init websocket connection
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, fmt.Sprintf("ws://%s:%s/api/websocket", ip, port), nil)
	if err != nil {
		ctxCancel()
		slog.Error("Failed to connect to websocket at ws://%s:%s/api/websocket. Check IP address and port\n", ip, port)
		return nil, nil, nil, err
	}

	// Read auth_required message
	_, err = ReadMessage(conn, ctx)
	if err != nil {
		ctxCancel()
		slog.Error("Unknown error creating websocket client\n")
		return nil, nil, nil, err
	}

	// Send auth message
	err = SendAuthMessage(conn, ctx, authToken)
	if err != nil {
		ctxCancel()
		slog.Error("Unknown error creating websocket client\n")
		return nil, nil, nil, err
	}

	// Verify auth message was successful
	err = VerifyAuthResponse(conn, ctx)
	if err != nil {
		ctxCancel()
		slog.Error("Auth token is invalid. Please double check it or create a new token in your Home Assistant profile\n")
		return nil, nil, nil, err
	}

	return conn, ctx, ctxCancel, nil
}

func SendAuthMessage(conn *websocket.Conn, ctx context.Context, token string) error {
	err := conn.WriteJSON(AuthMessage{MsgType: "auth", AccessToken: token})
	if err != nil {
		return err
	}
	return nil
}

type authResponse struct {
	MsgType string `json:"type"`
	Message string `json:"message"`
}

func VerifyAuthResponse(conn *websocket.Conn, ctx context.Context) error {
	msg, err := ReadMessage(conn, ctx)
	if err != nil {
		return err
	}

	var authResp authResponse
	json.Unmarshal(msg, &authResp)
	// log.Println(authResp.MsgType)
	if authResp.MsgType != "auth_ok" {
		return ErrInvalidToken
	}

	return nil
}

type SubEvent struct {
	Id        int64  `json:"id"`
	Type      string `json:"type"`
	EventType string `json:"event_type"`
}

func SubscribeToStateChangedEvents(id int64, conn *WebsocketWriter, ctx context.Context) {
	SubscribeToEventType("state_changed", conn, ctx, id)
}

func SubscribeToEventType(eventType string, conn *WebsocketWriter, ctx context.Context, id ...int64) {
	var finalId int64
	if len(id) == 0 {
		finalId = i.GetId()
	} else {
		finalId = id[0]
	}
	e := SubEvent{
		Id:        finalId,
		Type:      "subscribe_events",
		EventType: eventType,
	}
	err := conn.WriteMessage(e, ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("error writing to websocket: %w", err)
		slog.Error(wrappedErr.Error())
		panic(wrappedErr)
	}
	// m, _ := ReadMessage(conn, ctx)
	// log.Default().Println(string(m))
}
