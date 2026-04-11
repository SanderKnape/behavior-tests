package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"me/internal/platform/db"
	"me/internal/todos"
	"me/internal/users"
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

	r := setupRouter(database)
	log.Fatal(r.Run(":8080"))
}

func setupRouter(database todos.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})

	todos.RegisterRoutes(r, database)
	users.RegisterRoutes(r, database)

	return r
}
