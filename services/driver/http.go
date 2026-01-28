package driver

import "net/http"

func registerHTTP(s *Service, mux *http.ServeMux) {
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/drivers/", s.driverAPI)
	mux.HandleFunc("/ws/drivers/", s.driverWS)
}
