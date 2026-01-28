package driver

import (
	"context"
	"encoding/json"
	"strconv"
	"time"
)

func (s *Service) StartConsumers(ctx context.Context) {
	go s.consumeRideRequestsSupervisor(ctx)
}

func (s *Service) consumeRideRequestsSupervisor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ch, err := s.MQ.Consume("driver_matching")
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		for msg := range ch {
			var req struct {
				RideID         string `json:"ride_id"`
				RideType       string `json:"ride_type"`
				PickupLocation struct {
					Lat     float64 `json:"lat"`
					Lng     float64 `json:"lng"`
					Address string  `json:"address"`
				} `json:"pickup_location"`
				DestinationLocation struct {
					Lat     float64 `json:"lat"`
					Lng     float64 `json:"lng"`
					Address string  `json:"address"`
				} `json:"destination_location"`
				EstimatedFare float64 `json:"estimated_fare"`
				CorrelationID string  `json:"correlation_id"`
			}

			if err := json.Unmarshal(msg.Body, &req); err != nil {
				_ = msg.Nack(false, false)
				continue
			}
			if req.RideID == "" || req.RideType == "" {
				_ = msg.Nack(false, false)
				continue
			}

			cands, err := s.findCandidates(ctx, req.RideType, req.PickupLocation.Lat, req.PickupLocation.Lng)
			if err != nil {
				_ = msg.Nack(false, true)
				continue
			}

			// offer to top 3 drivers (Phase2)
			for i := 0; i < len(cands) && i < 3; i++ {
				offerID := "offer_" + req.RideID + "_" + strconv.Itoa(i)
				_ = s.Drivers.Send(cands[i].ID, map[string]any{
					"type":        "ride_offer",
					"offer_id":    offerID,
					"ride_id":     req.RideID,
					"ride_number": req.RideID, // optional; for demo
					"pickup_location": map[string]any{
						"latitude":  req.PickupLocation.Lat,
						"longitude": req.PickupLocation.Lng,
						"address":   req.PickupLocation.Address,
					},
					"destination_location": map[string]any{
						"latitude":  req.DestinationLocation.Lat,
						"longitude": req.DestinationLocation.Lng,
						"address":   req.DestinationLocation.Address,
					},
					"estimated_fare":        req.EstimatedFare,
					"distance_to_pickup_km": cands[i].Distance,
					"expires_at":            time.Now().UTC().Add(30 * time.Second).Format(time.RFC3339),
				})
			}

			_ = msg.Ack(false)
		}

		time.Sleep(300 * time.Millisecond)
	}
}
