package driver

import (
	"net/http"
	"strconv"

	"ride-hail-system/internal/config"
	"ride-hail-system/internal/logger"
	"ride-hail-system/internal/rabbit"
	"ride-hail-system/internal/websocket"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	Cfg config.Config
	DB  *pgxpool.Pool
	MQ  *rabbit.Client
	Log logger.Logger

	Drivers *websocket.Hub

	ServerHTTP *http.Server

	lastLoc *rateLimiter
}

func New(cfg config.Config, db *pgxpool.Pool, mq *rabbit.Client) *Service {
	s := &Service{
		Cfg:     cfg,
		DB:      db,
		MQ:      mq,
		Log:     logger.New("driver-location-service"),
		Drivers: websocket.NewHub(logger.New("driver-ws")),
		lastLoc: newRateLimiter(3),
	}
	mux := http.NewServeMux()
	registerHTTP(s, mux)
	s.ServerHTTP = &http.Server{Addr: ":" + strconv.Itoa(cfg.Services.DriverPort), Handler: mux}
	return s
}
