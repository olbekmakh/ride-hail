package websocket

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"ride-hail-system/internal/jwt"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type AuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func AuthenticateConn(ctx context.Context, conn *websocket.Conn, secret, role, mustUserID string) (jwt.Claims, error) {
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	var m AuthMessage
	if err := conn.ReadJSON(&m); err != nil {
		return jwt.Claims{}, err
	}
	if m.Type != "auth" {
		return jwt.Claims{}, errors.New("invalid auth type")
	}

	token := m.Token
	if !strings.HasPrefix(token, "Bearer ") {
		token = "Bearer " + token
	}

	c, err := jwt.ParseBearer(secret, token)
	if err != nil {
		return jwt.Claims{}, err
	}
	if err := jwt.RequireRole(c, role); err != nil {
		return jwt.Claims{}, err
	}
	if c.Subject != mustUserID {
		return jwt.Claims{}, errors.New("user mismatch")
	}

	// remove auth deadline
	_ = conn.SetReadDeadline(time.Time{})

	// keep alive: ping every 30s, close if no pong in 60s
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()

	return c, nil
}
