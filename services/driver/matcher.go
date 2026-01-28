package driver

import (
	"context"

	"ride-hail-system/internal/geo"
)

type candidate struct {
	ID       string
	Lat      float64
	Lng      float64
	Rating   float64
	Distance float64
}

func (s *Service) findCandidates(ctx context.Context, rideType string, pickLat, pickLng float64) ([]candidate, error) {
	// NOTE: postgis not required — we use haversine in Go
	rows, err := s.DB.Query(ctx, `
		SELECT d.id, d.rating, c.latitude, c.longitude
		FROM drivers d
		JOIN coordinates c
		  ON c.entity_id = d.id
		 AND c.entity_type = 'driver'
		 AND c.is_current = true
		WHERE d.status = 'AVAILABLE'
		  AND d.vehicle_type = $1
		LIMIT 100
	`, rideType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]candidate, 0, 16)
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.ID, &c.Rating, &c.Lat, &c.Lng); err != nil {
			continue
		}
		c.Distance = geo.HaversineKm(c.Lat, c.Lng, pickLat, pickLng)
		if c.Distance <= 5.0 {
			out = append(out, c)
		}
	}

	// sort by distance asc, rating desc (simple O(n^2) ok for top<=100)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			swap := false
			if out[j].Distance < out[i].Distance {
				swap = true
			} else if out[j].Distance == out[i].Distance && out[j].Rating > out[i].Rating {
				swap = true
			}
			if swap {
				out[i], out[j] = out[j], out[i]
			}
		}
	}

	return out, nil
}
