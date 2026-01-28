package logger

import (
	"encoding/json"
	"os"
	"runtime/debug"
	"time"
)

type Logger struct {
	Service  string
	Hostname string
}

func New(service string) Logger {
	hn, _ := os.Hostname()
	return Logger{Service: service, Hostname: hn}
}

func (l Logger) Info(action, msg, requestID, rideID string) {
	l.print("INFO", action, msg, requestID, rideID, nil)
}

func (l Logger) Debug(action, msg, requestID, rideID string) {
	l.print("DEBUG", action, msg, requestID, rideID, nil)
}

func (l Logger) Error(action, msg string, err error, requestID, rideID string) {
	var e any
	if err != nil {
		e = map[string]any{
			"msg":   err.Error(),
			"stack": string(debug.Stack()),
		}
	}
	l.print("ERROR", action, msg, requestID, rideID, e)
}

func (l Logger) print(level, action, msg, requestID, rideID string, err any) {
	m := map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"level":      level,
		"service":    l.Service,
		"action":     action,
		"message":    msg,
		"hostname":   l.Hostname,
		"request_id": requestID,
		"ride_id":    rideID,
	}
	if err != nil {
		m["error"] = err
	}
	b, _ := json.Marshal(m)
	os.Stdout.Write(append(b, '\n'))
}
