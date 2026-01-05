package main

import (
	"auth/internal/data"
	"auth/migrations"
	"database/sql"
	"log"
	"log/slog"
	"os"
	"testing"

	"github.com/nats-io/nats.go"
)

var app *application
var dsn string

func TestMain(m *testing.M) {

	dsn = os.Getenv("AUTH_DB_TEST_DSN")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("failed to open db connection", slog.Any("err", err))
	}

	if err := migrations.RunUpMigrations(dsn); err != nil {
		log.Fatal("failed to run db migrations", slog.Any("err", err))
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	opts := nats.Options{}
	if url := os.Getenv("NATS_URL"); url != "" {
		opts.Url = url
	}
	nc, err := opts.Connect()
	if err != nil {
		log.Fatal("failed to connect to NATS", slog.Any("err", err))
	}

	app = &application{
		nc:              nc,
		logger:          logger,
		models:          data.NewModels(db),
		jwtAccessSecret: "test-secret-ensure-32-bytes-long-string!",
	}

	err = app.start()
	if err != nil {
		log.Fatal("failed to start the service", slog.Any("err", err))
	}

	c := m.Run()
	nc.Close()
	os.Exit(c)
}
