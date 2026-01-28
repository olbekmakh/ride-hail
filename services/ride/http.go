package ride

import "net/http"

func registerHTTP(s *Service, mux *http.ServeMux) {
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/rides", s.rides)
	mux.HandleFunc("/rides/", s.rideActions)
	mux.HandleFunc("/ws/passengers/", s.passengerWS)
}
