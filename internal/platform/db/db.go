package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sort"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed seeds/*.sql
var seedsFS embed.FS

func Open() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL environment variable is required")
	}

	database, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	return database, nil
}

func RunMigrations(database *sql.DB) error {
	return runMigrations(database, migrationsFS)
}

func runMigrations(database *sql.DB, fsys fs.FS) error {
	src, err := iofs.New(fsys, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	driver, err := postgres.WithInstance(database, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}

func RunSeeds(database *sql.DB) error {
	return runSeeds(database, seedsFS)
}

func runSeeds(database *sql.DB, fsys fs.FS) error {
	entries, err := fs.ReadDir(fsys, "seeds")
	if err != nil {
		return fmt.Errorf("reading seeds: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		data, err := fs.ReadFile(fsys, "seeds/"+entry.Name())
		if err != nil {
			return fmt.Errorf("reading seed %s: %w", entry.Name(), err)
		}

		if _, err := database.Exec(string(data)); err != nil {
			return fmt.Errorf("executing seed %s: %w", entry.Name(), err)
		}

		log.Printf("applied seed: %s", entry.Name())
	}

	return nil
}
