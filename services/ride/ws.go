package ride

import (
	"net/http"
	"strings"

	"ride-hail-system/internal/websocket"
)

func (s *Service) passengerWS(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "ws" || parts[1] != "passengers" {
		http.NotFound(w, r)
		return
	}
	passengerID := parts[2]

	conn, err := websocket.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	_, err = websocket.AuthenticateConn(r.Context(), conn, s.Cfg.JWT.Secret, "PASSENGER", passengerID)
	if err != nil {
		return
	}

	s.Passengers.Set(passengerID, conn)
	defer s.Passengers.Del(passengerID)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
