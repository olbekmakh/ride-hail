package config

import (
	"os"
	"strconv"
)

type DB struct {
	Host string
	Port int
	User string
	Pass string
	Name string
}

type Rabbit struct {
	Host string
	Port int
	User string
	Pass string
}

type Services struct {
	RidePort   int
	DriverPort int
	AdminPort  int
}

type JWT struct {
	Secret string
}

type Config struct {
	DB       DB
	Rabbit   Rabbit
	Services Services
	JWT      JWT
}

func Load() Config {
	return Config{
		DB: DB{
			Host: getenv("DB_HOST", "localhost"),
			Port: mustInt(getenv("DB_PORT", "5432")),
			User: getenv("DB_USER", "ridehail_user"),
			Pass: getenv("DB_PASSWORD", "ridehail_pass"),
			Name: getenv("DB_NAME", "ridehail_db"),
		},
		Rabbit: Rabbit{
			Host: getenv("RABBITMQ_HOST", "localhost"),
			Port: mustInt(getenv("RABBITMQ_PORT", "5672")),
			User: getenv("RABBITMQ_USER", "guest"),
			Pass: getenv("RABBITMQ_PASSWORD", "guest"),
		},
		Services: Services{
			RidePort:   mustInt(getenv("RIDE_SERVICE_PORT", "3000")),
			DriverPort: mustInt(getenv("DRIVER_LOCATION_SERVICE_PORT", "3001")),
			AdminPort:  mustInt(getenv("ADMIN_SERVICE_PORT", "3004")),
		},
		JWT: JWT{
			Secret: getenv("JWT_SECRET", "dev_secret_change_me"),
		},
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func mustInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
