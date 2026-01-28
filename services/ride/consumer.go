package ride

import (
	"context"
	"encoding/json"
	"time"
)

// StartConsumers запускается из main.go
func (s *Service) StartConsumers(ctx context.Context) {
	go s.consumeDriverResponses(ctx)
	go s.consumeRideStatus(ctx)
	go s.consumeLocationUpdates(ctx)
}

// -------- driver.response.{ride_id}
func (s *Service) consumeDriverResponses(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ch, err := s.MQ.Consume("driver_responses")
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		for msg := range ch {
			var p struct {
				RideID        string `json:"ride_id"`
				DriverID      string `json:"driver_id"`
				Accepted      bool   `json:"accepted"`
				CorrelationID string `json:"correlation_id"`
			}

			if err := json.Unmarshal(msg.Body, &p); err != nil {
				_ = msg.Nack(false, false)
				continue
			}

			if p.Accepted {
				ct, _ := s.DB.Exec(ctx, `
					UPDATE rides
					SET status='MATCHED', driver_id=$2, matched_at=NOW(), updated_at=NOW()
					WHERE id=$1 AND status='REQUESTED' AND driver_id IS NULL
				`, p.RideID, p.DriverID)

				if ct.RowsAffected() > 0 {
					var passengerID string
					_ = s.DB.QueryRow(ctx,
						`SELECT passenger_id FROM rides WHERE id=$1`, p.RideID,
					).Scan(&passengerID)

					_ = s.Passengers.Send(passengerID, map[string]any{
						"type":           "ride_status_update",
						"ride_id":        p.RideID,
						"status":         "MATCHED",
						"driver_id":      p.DriverID,
						"correlation_id": p.CorrelationID,
					})

					_ = s.PublishRideStatus(ctx, p.CorrelationID, p.RideID, "MATCHED", p.DriverID)
				}
			}

			_ = msg.Ack(false)
		}
	}
}

// -------- ride.status.*
func (s *Service) consumeRideStatus(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ch, err := s.MQ.Consume("ride_status")
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		for msg := range ch {
			var p struct {
				RideID string `json:"ride_id"`
				Status string `json:"status"`
			}

			if err := json.Unmarshal(msg.Body, &p); err != nil {
				_ = msg.Nack(false, false)
				continue
			}

			var passengerID string
			_ = s.DB.QueryRow(ctx,
				`SELECT passenger_id FROM rides WHERE id=$1`, p.RideID,
			).Scan(&passengerID)

			_ = s.Passengers.Send(passengerID, map[string]any{
				"type":    "ride_status_update",
				"ride_id": p.RideID,
				"status":  p.Status,
			})

			_ = msg.Ack(false)
		}
	}
}

// -------- location_updates_ride
func (s *Service) consumeLocationUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ch, err := s.MQ.Consume("location_updates_ride")
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		for msg := range ch {
			var p struct {
				RideID   string `json:"ride_id"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			}

			if err := json.Unmarshal(msg.Body, &p); err != nil {
				_ = msg.Nack(false, false)
				continue
			}

			var passengerID string
			_ = s.DB.QueryRow(ctx,
				`SELECT passenger_id FROM rides WHERE id=$1`, p.RideID,
			).Scan(&passengerID)

			_ = s.Passengers.Send(passengerID, map[string]any{
				"type":    "driver_location_update",
				"ride_id": p.RideID,
				"driver_location": map[string]any{
					"lat": p.Location.Lat,
					"lng": p.Location.Lng,
				},
			})

			_ = msg.Ack(false)
		}
	}
}
