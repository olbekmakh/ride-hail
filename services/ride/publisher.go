package ride

import (
	"context"
	"strings"
	"time"
)

// PublishRideRequest → ride_topic → ride.request.{ride_type}
func (s *Service) PublishRideRequest(
	ctx context.Context,
	reqID string,
	rideID string,
	rideNumber string,
	rideType string,
	passengerID string,
	pickLat, pickLng float64,
	pickAddr string,
	dstLat, dstLng float64,
	dstAddr string,
	estimatedFare float64,
) error {
	routingKey := "ride.request." + strings.ToLower(rideType)

	payload := map[string]any{
		"ride_id":      rideID,
		"ride_number":  rideNumber,
		"ride_type":    rideType,
		"passenger_id": passengerID,
		"pickup_location": map[string]any{
			"lat":     pickLat,
			"lng":     pickLng,
			"address": pickAddr,
		},
		"destination_location": map[string]any{
			"lat":     dstLat,
			"lng":     dstLng,
			"address": dstAddr,
		},
		"estimated_fare":  estimatedFare,
		"timeout_seconds": 120,
		"correlation_id":  reqID,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	return s.MQ.PublishJSON(ctx, "ride_topic", routingKey, payload)
}

// PublishRideStatus → ride_topic → ride.status.{status}
func (s *Service) PublishRideStatus(
	ctx context.Context,
	reqID string,
	rideID string,
	status string,
	driverID string,
) error {
	routingKey := "ride.status." + strings.ToLower(status)

	payload := map[string]any{
		"ride_id":        rideID,
		"status":         status,
		"driver_id":      driverID,
		"correlation_id": reqID,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	return s.MQ.PublishJSON(ctx, "ride_topic", routingKey, payload)
}
