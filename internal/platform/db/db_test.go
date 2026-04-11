package db

import (
	"errors"
	"testing"
	"testing/fstest"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestOpen_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := Open()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is not set")
	}
}

func TestRunSeeds_ExecutesSeedsInSortedOrder(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close() //nolint:errcheck

	fsys := fstest.MapFS{
		"seeds/002_second.sql": {Data: []byte("SELECT 2")},
		"seeds/001_first.sql":  {Data: []byte("SELECT 1")},
	}

	// Expect first file (001) before second (002) — sort order verified
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT 2").WillReturnResult(sqlmock.NewResult(0, 0))

	if err := runSeeds(mockDB, fsys); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestRunSeeds_ReturnsErrorOnExecFailure(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close() //nolint:errcheck

	fsys := fstest.MapFS{
		"seeds/001_first.sql": {Data: []byte("SELECT 1")},
	}

	mock.ExpectExec("SELECT 1").WillReturnError(errors.New("db error"))

	if err := runSeeds(mockDB, fsys); err == nil {
		t.Fatal("expected error on exec failure")
	}
}

func TestRunSeeds_ReturnsErrorForMissingSeedsDir(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close() //nolint:errcheck

	if err := runSeeds(mockDB, fstest.MapFS{}); err == nil {
		t.Fatal("expected error when seeds directory is missing")
	}
}

func TestRunMigrations_ReturnsErrorForMissingMigrationsDir(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close() //nolint:errcheck

	if err := runMigrations(mockDB, fstest.MapFS{}); err == nil {
		t.Fatal("expected error when migrations directory is missing")
	}
}
