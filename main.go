package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-hail-system/internal/config"
	"ride-hail-system/internal/logger"
	"ride-hail-system/internal/postgres"
	"ride-hail-system/internal/rabbit"
	"ride-hail-system/services/admin"
	"ride-hail-system/services/driver"
	"ride-hail-system/services/ride"
)

func main() {
	cfg := config.Load()
	log := logger.New("main")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		log.Error("db_connect_failed", "db connect failed", err, "", "")
		os.Exit(1)
	}
	defer db.Close()

	rmqURL := fmt.Sprintf("amqp://%s:%s@%s:%d/", cfg.Rabbit.User, cfg.Rabbit.Pass, cfg.Rabbit.Host, cfg.Rabbit.Port)
	mq := rabbit.New(rmqURL, logger.New("rabbit"))
	if err := mq.ConnectAndSetup(ctx); err != nil {
		log.Error("mq_connect_failed", "mq connect failed", err, "", "")
		os.Exit(1)
	}

	rideSvc := ride.New(cfg, db, mq)
	driverSvc := driver.New(cfg, db, mq)
	adminSvc := admin.New(cfg, db)

	rideSvc.StartConsumers(ctx)
	driverSvc.StartConsumers(ctx)

	go serve(log, "ride-service", rideSvc.ServerHTTP)
	go serve(log, "driver-location-service", driverSvc.ServerHTTP)
	go serve(log, "admin-service", adminSvc.ServerHTTP)

	<-ctx.Done()

	shutdown(log, rideSvc.ServerHTTP)
	shutdown(log, driverSvc.ServerHTTP)
	shutdown(log, adminSvc.ServerHTTP)
}

func serve(l logger.Logger, name string, srv *http.Server) {
	l.Info("http_start", name+" listening on "+srv.Addr, "", "")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		l.Error("http_error", name+" server error", err, "", "")
	}
}

func shutdown(l logger.Logger, srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	l.Info("http_stop", "server stopped "+srv.Addr, "", "")
}
