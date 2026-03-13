package testutils

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)
import "auth/migrations"

func ResetTestDB(t *testing.T, dsn string) {
	t.Helper()
	d, err := iofs.New(migrations.MigrationFiles, ".")
	if err != nil {
		t.Fatalf("failed to read migration files: %v", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		t.Fatalf("migration init failed: %v", err)
	}

	if err := m.Drop(); err != nil {
		t.Fatalf("drop failed: %v", err)
	}

	if err := migrations.RunUpMigrations(dsn); err != nil {
		t.Fatalf("up failed: %v", err)
	}
}
