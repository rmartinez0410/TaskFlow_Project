package main

import (
	"auth/internal/data"
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
)

type application struct {
	nc              *nats.Conn
	js              nats.JetStreamContext
	logger          *slog.Logger
	models          *data.Models
	jwtAccessSecret string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := godotenv.Load(); err != nil {
		logger.Warn("no .env file found")
	}

	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	if accessSecret == "" {
		logger.Error("failed to get jwt secret")
		os.Exit(1)
	}

	opts := nats.Options{
		AllowReconnect: true,
		MaxReconnect:   5,
		ReconnectWait:  5 * time.Second,
		Timeout:        time.Second,
	}

	nc, err := opts.Connect()
	if err != nil {
		logger.Error("failed to connect to NATS", err.Error())
		os.Exit(1)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		logger.Error("jetstream not available", err.Error())
		os.Exit(1)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:        "auth",
		Description: "stream for authentication commands and events",
		Subjects:    []string{"auth.>"},
		Retention:   nats.WorkQueuePolicy,
		MaxAge:      7 * 24 * time.Hour,
		MaxMsgs:     1000000,
		Discard:     nats.DiscardOld,
	})

	db, err := openDB(os.Getenv("AUTH_DB_DSN"))
	if err != nil {
		logger.Error("failed to connect to database", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("error during database closure", err.Error())
			os.Exit(1)
		}
	}()

	app := &application{
		nc:              nc,
		js:              js,
		logger:          logger,
		models:          data.NewModels(db),
		jwtAccessSecret: accessSecret,
	}

	err = app.start()
	if err != nil {
		app.logger.Error("failed to start application", err.Error())
		os.Exit(1)
	}

	logger.Info("auth service started")
	select {}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
