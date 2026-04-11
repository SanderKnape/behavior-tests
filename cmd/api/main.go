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

	database := db.Open()
	defer func() { _ = database.Close() }()

	db.RunMigrations(database)

	if *seed {
		db.RunSeeds(database)
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
