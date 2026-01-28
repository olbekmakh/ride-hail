package ride

import (
	"net/http"
	"strings"
	"time"

	"ride-hail-system/internal/geo"
	"ride-hail-system/internal/httpx"
	"ride-hail-system/internal/jwt"
)

func (s *Service) health(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, 200, map[string]any{"ok": true, "service": "ride-service"})
}

type createRideReq struct {
	PassengerID          string  `json:"passenger_id"`
	PickupLatitude       float64 `json:"pickup_latitude"`
	PickupLongitude      float64 `json:"pickup_longitude"`
	PickupAddress        string  `json:"pickup_address"`
	DestinationLatitude  float64 `json:"destination_latitude"`
	DestinationLongitude float64 `json:"destination_longitude"`
	DestinationAddress   string  `json:"destination_address"`
	RideType             string  `json:"ride_type"`
}

func (s *Service) rides(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	reqID := httpx.ReqID(r)

	claims, err := jwt.ParseBearer(s.Cfg.JWT.Secret, r.Header.Get("Authorization"))
	if err != nil || jwt.RequireRole(claims, "PASSENGER") != nil {
		http.Error(w, "unauthorized", 401)
		return
	}

	var body createRideReq
	if err := httpx.ReadJSON(r, &body); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	if body.PassengerID != claims.Subject {
		http.Error(w, "passenger mismatch", 403)
		return
	}

	distKm := geo.HaversineKm(body.PickupLatitude, body.PickupLongitude, body.DestinationLatitude, body.DestinationLongitude)
	durationMin := int(distKm/40*60 + 0.5)
	if durationMin < 1 {
		durationMin = 1
	}
	fare := estimateFare(body.RideType, distKm, float64(durationMin))
	rideNumber := "RIDE_" + time.Now().UTC().Format("20060102_150405") + "_001"

	ctx := r.Context()
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "server error", 500)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var pickupID, destID, rideID string

	err = tx.QueryRow(ctx, `
		INSERT INTO coordinates(entity_id,entity_type,address,latitude,longitude,is_current,fare_amount,distance_km,duration_minutes)
		VALUES($1,'passenger',$2,$3,$4,true,$5,$6,$7) RETURNING id
	`, body.PassengerID, body.PickupAddress, body.PickupLatitude, body.PickupLongitude, fare, distKm, durationMin).Scan(&pickupID)
	if err != nil {
		http.Error(w, "server error", 500)
		return
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO coordinates(entity_id,entity_type,address,latitude,longitude,is_current)
		VALUES($1,'passenger',$2,$3,$4,false) RETURNING id
	`, body.PassengerID, body.DestinationAddress, body.DestinationLatitude, body.DestinationLongitude).Scan(&destID)
	if err != nil {
		http.Error(w, "server error", 500)
		return
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO rides(ride_number,passenger_id,vehicle_type,status,estimated_fare,pickup_coordinate_id,destination_coordinate_id)
		VALUES($1,$2,$3,'REQUESTED',$4,$5,$6) RETURNING id
	`, rideNumber, body.PassengerID, body.RideType, fare, pickupID, destID).Scan(&rideID)
	if err != nil {
		http.Error(w, "server error", 500)
		return
	}

	_, _ = tx.Exec(ctx, `INSERT INTO ride_events(ride_id,event_type,event_data) VALUES($1,'RIDE_REQUESTED',$2::jsonb)`,
		rideID, `{"status":"REQUESTED"}`)

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "server error", 500)
		return
	}

	_ = s.MQ.PublishJSON(ctx, "ride_topic", "ride.request."+strings.ToLower(body.RideType), map[string]any{
		"ride_id":      rideID,
		"ride_number":  rideNumber,
		"ride_type":    body.RideType,
		"passenger_id": body.PassengerID,
		"pickup_location": map[string]any{
			"lat": body.PickupLatitude, "lng": body.PickupLongitude, "address": body.PickupAddress,
		},
		"correlation_id": reqID,
	})

	s.Log.Info("ride_requested", "ride created", reqID, rideID)

	httpx.WriteJSON(w, 201, map[string]any{
		"ride_id":                    rideID,
		"ride_number":                rideNumber,
		"status":                     "REQUESTED",
		"estimated_fare":             fare,
		"estimated_duration_minutes": durationMin,
		"estimated_distance_km":      round2(distKm),
	})
}

func (s *Service) rideActions(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "rides" || parts[2] != "cancel" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	rideID := parts[1]
	reqID := httpx.ReqID(r)

	claims, err := jwt.ParseBearer(s.Cfg.JWT.Secret, r.Header.Get("Authorization"))
	if err != nil || jwt.RequireRole(claims, "PASSENGER") != nil {
		http.Error(w, "unauthorized", 401)
		return
	}

	_, _ = s.DB.Exec(r.Context(), `
		UPDATE rides SET status='CANCELLED', cancelled_at=NOW(), updated_at=NOW()
		WHERE id=$1 AND passenger_id=$2 AND status <> 'COMPLETED'
	`, rideID, claims.Subject)

	_ = s.MQ.PublishJSON(r.Context(), "ride_topic", "ride.status.cancelled", map[string]any{
		"ride_id": rideID, "status": "CANCELLED", "correlation_id": reqID,
	})

	_ = s.Passengers.Send(claims.Subject, map[string]any{
		"type": "ride_status_update", "ride_id": rideID, "status": "CANCELLED", "correlation_id": reqID,
	})

	httpx.WriteJSON(w, 200, map[string]any{"ride_id": rideID, "status": "CANCELLED"})
}

func estimateFare(rideType string, distanceKm, durationMin float64) float64 {
	base, perKm, perMin := 500.0, 100.0, 50.0
	if rideType == "PREMIUM" {
		base, perKm, perMin = 800, 120, 60
	}
	if rideType == "XL" {
		base, perKm, perMin = 1000, 150, 75
	}
	return round2(base + distanceKm*perKm + durationMin*perMin)
}
func round2(x float64) float64 { return float64(int(x*100+0.5)) / 100 }
