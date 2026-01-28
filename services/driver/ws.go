package driver

import (
	"encoding/json"
	"net/http"
	"strings"

	"ride-hail-system/internal/websocket"
)

func (s *Service) driverWS(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "ws" || parts[1] != "drivers" {
		http.NotFound(w, r)
		return
	}
	driverID := parts[2]

	conn, err := websocket.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	_, err = websocket.AuthenticateConn(r.Context(), conn, s.Cfg.JWT.Secret, "DRIVER", driverID)
	if err != nil {
		return
	}

	s.Drivers.Set(driverID, conn)
	defer s.Drivers.Del(driverID)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// expected:
		// {"type":"ride_response","offer_id":"...","ride_id":"...","accepted":true}
		var msg struct {
			Type     string `json:"type"`
			OfferID  string `json:"offer_id"`
			RideID   string `json:"ride_id"`
			Accepted bool   `json:"accepted"`
		}
		_ = json.Unmarshal(data, &msg)
		if msg.Type != "ride_response" || msg.RideID == "" {
			continue
		}

		_ = s.MQ.PublishJSON(r.Context(), "driver_topic", "driver.response."+msg.RideID, map[string]any{
			"ride_id":        msg.RideID,
			"driver_id":      driverID,
			"accepted":       msg.Accepted,
			"correlation_id": "ws_" + driverID,
		})
	}
}
