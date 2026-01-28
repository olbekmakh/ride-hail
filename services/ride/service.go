package ride

import (
	"net/http"

	"ride-hail-system/internal/config"
	"ride-hail-system/internal/logger"
	"ride-hail-system/internal/postgres"
	"ride-hail-system/internal/rabbit"
	"ride-hail-system/internal/websocket"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	Cfg config.Config
	DB  *pgxpool.Pool
	MQ  *rabbit.Client
	Log logger.Logger

	Passengers *websocket.Hub
	ServerHTTP *http.Server
}

func New(cfg config.Config, db *pgxpool.Pool, mq *rabbit.Client) *Service {
	s := &Service{
		Cfg:        cfg,
		DB:         db,
		MQ:         mq,
		Log:        logger.New("ride-service"),
		Passengers: websocket.NewHub(logger.New("ride-ws")),
	}
	mux := http.NewServeMux()
	registerHTTP(s, mux)
	s.ServerHTTP = &http.Server{Addr: ":" + itoa(cfg.Services.RidePort), Handler: mux}
	return s
}

func itoa(n int) string { return postgres.Itoa(n) }
