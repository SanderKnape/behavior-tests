package main

import (
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"

	"me/cmd/api/server"
	"me/internal/platform/db"
)

func main() {
	seed := flag.Bool("seed", false, "seed the database with test data and exit")
	flag.Parse()

	_ = godotenv.Load()

	database, err := db.Open()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer func() { _ = database.Close() }()

	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	if *seed {
		if err := db.RunSeeds(database); err != nil {
			log.Fatalf("seeds: %v", err)
		}
		return
	}

	r := server.New(database)
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Fatal(r.Run(addr))
}
