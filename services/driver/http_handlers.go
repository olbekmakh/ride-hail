package driver

import (
	"net/http"
	"strings"
	"time"

	"ride-hail-system/internal/httpx"
	"ride-hail-system/internal/jwt"
)

func (s *Service) health(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, 200, map[string]any{"ok": true, "service": "driver-location-service"})
}

func (s *Service) driverAPI(w http.ResponseWriter, r *http.Request) {
	// /drivers/{driver_id}/{action}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "drivers" {
		http.NotFound(w, r)
		return
	}
	driverID := parts[1]
	action := parts[2]

	claims, err := jwt.ParseBearer(s.Cfg.JWT.Secret, r.Header.Get("Authorization"))
	if err != nil || jwt.RequireRole(claims, "DRIVER") != nil || claims.Subject != driverID {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch action {
	case "online":
		if r.Method == http.MethodPost {
			s.goOnline(w, r, driverID)
			return
		}
	case "offline":
		if r.Method == http.MethodPost {
			s.goOffline(w, r, driverID)
			return
		}
	case "location":
		if r.Method == http.MethodPost {
			s.updateLocation(w, r, driverID)
			return
		}
	case "arrived":
		if r.Method == http.MethodPost {
			s.arrived(w, r, driverID)
			return
		}
	case "start":
		if r.Method == http.MethodPost {
			s.startRide(w, r, driverID)
			return
		}
	case "complete":
		if r.Method == http.MethodPost {
			s.completeRide(w, r, driverID)
			return
		}
	}

	http.NotFound(w, r)
}

func (s *Service) goOnline(w http.ResponseWriter, r *http.Request, driverID string) {
	type body struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	var b body
	if err := httpx.ReadJSON(r, &b); err != nil || !validCoord(b.Latitude, b.Longitude) {
		http.Error(w, "bad request", 400)
		return
	}

	_, _ = s.DB.Exec(r.Context(), `UPDATE drivers SET status='AVAILABLE', updated_at=NOW() WHERE id=$1`, driverID)

	// ensure current coordinates for driver (insert)
	_, _ = s.DB.Exec(r.Context(), `
		UPDATE coordinates SET is_current=false, updated_at=NOW()
		WHERE entity_id=$1 AND entity_type='driver' AND is_current=true
	`, driverID)

	_, _ = s.DB.Exec(r.Context(), `
		INSERT INTO coordinates(entity_id,entity_type,address,latitude,longitude,is_current)
		VALUES($1,'driver','Online location',$2,$3,true)
	`, driverID, b.Latitude, b.Longitude)

	httpx.WriteJSON(w, 200, map[string]any{"status": "AVAILABLE", "message": "You are now online"})
}

func (s *Service) goOffline(w http.ResponseWriter, r *http.Request, driverID string) {
	_, _ = s.DB.Exec(r.Context(), `UPDATE drivers SET status='OFFLINE', updated_at=NOW() WHERE id=$1`, driverID)
	httpx.WriteJSON(w, 200, map[string]any{"status": "OFFLINE"})
}

func (s *Service) updateLocation(w http.ResponseWriter, r *http.Request, driverID string) {
	type body struct {
		Latitude       float64 `json:"latitude"`
		Longitude      float64 `json:"longitude"`
		AccuracyMeters float64 `json:"accuracy_meters"`
		SpeedKmh       float64 `json:"speed_kmh"`
		Heading        float64 `json:"heading_degrees"`
		RideID         string  `json:"ride_id,omitempty"`
	}
	var b body
	if err := httpx.ReadJSON(r, &b); err != nil || !validCoord(b.Latitude, b.Longitude) {
		http.Error(w, "bad request", 400)
		return
	}

	if !s.lastLoc.Allow(driverID) {
		http.Error(w, "rate limited", 429)
		return
	}

	_, _ = s.DB.Exec(r.Context(), `
		UPDATE coordinates SET is_current=false, updated_at=NOW()
		WHERE entity_id=$1 AND entity_type='driver' AND is_current=true
	`, driverID)

	var coordID string
	_ = s.DB.QueryRow(r.Context(), `
		INSERT INTO coordinates(entity_id,entity_type,address,latitude,longitude,is_current)
		VALUES($1,'driver','Current Location',$2,$3,true) RETURNING id
	`, driverID, b.Latitude, b.Longitude).Scan(&coordID)

	_ = s.MQ.PublishJSON(r.Context(), "location_fanout", "", map[string]any{
		"driver_id": driverID,
		"ride_id":   b.RideID,
		"location":  map[string]any{"lat": b.Latitude, "lng": b.Longitude},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})

	httpx.WriteJSON(w, 200, map[string]any{"coordinate_id": coordID})
}

func (s *Service) arrived(w http.ResponseWriter, r *http.Request, driverID string) {
	type body struct {
		RideID string `json:"ride_id"`
	}
	var b body
	if err := httpx.ReadJSON(r, &b); err != nil || b.RideID == "" {
		http.Error(w, "bad request", 400)
		return
	}
	_ = s.MQ.PublishJSON(r.Context(), "ride_topic", "ride.status.arrived", map[string]any{
		"ride_id": b.RideID, "status": "ARRIVED", "driver_id": driverID,
		"timestamp": time.Now().UTC().Format(time.RFC3339), "correlation_id": "http_" + driverID,
	})
	httpx.WriteJSON(w, 200, map[string]any{"ride_id": b.RideID, "status": "ARRIVED_SENT"})
}

func (s *Service) startRide(w http.ResponseWriter, r *http.Request, driverID string) {
	type body struct {
		RideID string `json:"ride_id"`
	}
	var b body
	if err := httpx.ReadJSON(r, &b); err != nil || b.RideID == "" {
		http.Error(w, "bad request", 400)
		return
	}

	_, _ = s.DB.Exec(r.Context(), `UPDATE drivers SET status='BUSY', updated_at=NOW() WHERE id=$1`, driverID)

	_ = s.MQ.PublishJSON(r.Context(), "ride_topic", "ride.status.in_progress", map[string]any{
		"ride_id": b.RideID, "status": "IN_PROGRESS", "driver_id": driverID,
		"timestamp": time.Now().UTC().Format(time.RFC3339), "correlation_id": "http_" + driverID,
	})
	httpx.WriteJSON(w, 200, map[string]any{"ride_id": b.RideID, "status": "BUSY"})
}

func (s *Service) completeRide(w http.ResponseWriter, r *http.Request, driverID string) {
	type body struct {
		RideID            string  `json:"ride_id"`
		ActualDistanceKm  float64 `json:"actual_distance_km"`
		ActualDurationMin int     `json:"actual_duration_minutes"`
	}
	var b body
	if err := httpx.ReadJSON(r, &b); err != nil || b.RideID == "" {
		http.Error(w, "bad request", 400)
		return
	}

	_, _ = s.DB.Exec(r.Context(), `UPDATE drivers SET status='AVAILABLE', updated_at=NOW() WHERE id=$1`, driverID)

	_ = s.MQ.PublishJSON(r.Context(), "ride_topic", "ride.status.completed", map[string]any{
		"ride_id": b.RideID, "status": "COMPLETED", "driver_id": driverID,
		"distance_km": b.ActualDistanceKm, "duration_minutes": b.ActualDurationMin,
		"timestamp": time.Now().UTC().Format(time.RFC3339), "correlation_id": "http_" + driverID,
	})

	httpx.WriteJSON(w, 200, map[string]any{"ride_id": b.RideID, "status": "AVAILABLE"})
}

func validCoord(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}
