package admin

import (
	"net/http"
	"time"

	"ride-hail-system/internal/httpx"
	"ride-hail-system/internal/jwt"
)

func registerHTTP(s *Service, mux *http.ServeMux) {
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/admin/overview", s.overview)
	mux.HandleFunc("/admin/rides/active", s.activeRides)
}

func (s *Service) health(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, 200, map[string]any{"ok": true, "service": "admin-service"})
}

func (s *Service) authAdmin(w http.ResponseWriter, r *http.Request) (jwt.Claims, bool) {
	c, err := jwt.ParseBearer(s.Cfg.JWT.Secret, r.Header.Get("Authorization"))
	if err != nil || jwt.RequireRole(c, "ADMIN") != nil {
		http.Error(w, "unauthorized", 401)
		return jwt.Claims{}, false
	}
	return c, true
}

func (s *Service) overview(w http.ResponseWriter, r *http.Request) {
	_, ok := s.authAdmin(w, r)
	if !ok {
		return
	}

	var activeRides int
	var availableDrivers int
	var busyDrivers int

	_ = s.DB.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM rides WHERE status IN ('REQUESTED','MATCHED','EN_ROUTE','ARRIVED','IN_PROGRESS')
	`).Scan(&activeRides)

	_ = s.DB.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM drivers WHERE status='AVAILABLE'
	`).Scan(&availableDrivers)

	_ = s.DB.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM drivers WHERE status='BUSY'
	`).Scan(&busyDrivers)

	httpx.WriteJSON(w, 200, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"metrics": map[string]any{
			"active_rides":      activeRides,
			"available_drivers": availableDrivers,
			"busy_drivers":      busyDrivers,
		},
	})
}

func (s *Service) activeRides(w http.ResponseWriter, r *http.Request) {
	_, ok := s.authAdmin(w, r)
	if !ok {
		return
	}

	rows, err := s.DB.Query(r.Context(), `
		SELECT id, ride_number, status, passenger_id, driver_id, created_at
		FROM rides
		WHERE status IN ('REQUESTED','MATCHED','EN_ROUTE','ARRIVED','IN_PROGRESS')
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err != nil {
		http.Error(w, "server error", 500)
		return
	}
	defer rows.Close()

	out := make([]map[string]any, 0, 32)
	for rows.Next() {
		var id, num, status, passengerID string
		var driverID *string
		var createdAt time.Time
		if err := rows.Scan(&id, &num, &status, &passengerID, &driverID, &createdAt); err != nil {
			continue
		}
		m := map[string]any{
			"ride_id":      id,
			"ride_number":  num,
			"status":       status,
			"passenger_id": passengerID,
			"created_at":   createdAt.UTC().Format(time.RFC3339),
		}
		if driverID != nil {
			m["driver_id"] = *driverID
		}
		out = append(out, m)
	}

	httpx.WriteJSON(w, 200, map[string]any{"rides": out})
}
