package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"me/internal/todos"
	"me/internal/users"
)

func New(db todos.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})

	todos.RegisterRoutes(r, db)
	users.RegisterRoutes(r, db)

	return r
}
