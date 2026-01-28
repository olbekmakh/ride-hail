package admin

import (
	"net/http"
	"strconv"

	"ride-hail-system/internal/config"
	"ride-hail-system/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	Cfg config.Config
	DB  *pgxpool.Pool
	Log logger.Logger

	ServerHTTP *http.Server
}

func New(cfg config.Config, db *pgxpool.Pool) *Service {
	s := &Service{
		Cfg: cfg,
		DB:  db,
		Log: logger.New("admin-service"),
	}
	mux := http.NewServeMux()
	registerHTTP(s, mux)
	s.ServerHTTP = &http.Server{Addr: ":" + strconv.Itoa(cfg.Services.AdminPort), Handler: mux}
	return s
}
