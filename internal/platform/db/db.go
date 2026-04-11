package db

import (
	"database/sql"
	"embed"
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

func Open() *sql.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	database, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	if err := database.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	return database
}

func RunMigrations(database *sql.DB) {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		log.Fatalf("migration source: %v", err)
	}

	driver, err := postgres.WithInstance(database, &postgres.Config{})
	if err != nil {
		log.Fatalf("migration driver: %v", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		log.Fatalf("migrator: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("running migrations: %v", err)
	}
}

func RunSeeds(database *sql.DB) {
	entries, err := fs.ReadDir(seedsFS, "seeds")
	if err != nil {
		log.Fatalf("reading seeds: %v", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		data, err := seedsFS.ReadFile("seeds/" + entry.Name())
		if err != nil {
			log.Fatalf("reading seed %s: %v", entry.Name(), err)
		}

		if _, err := database.Exec(string(data)); err != nil {
			log.Fatalf("executing seed %s: %v", entry.Name(), err)
		}

		log.Printf("applied seed: %s", entry.Name())
	}
}
