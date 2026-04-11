package users

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB is the subset of *sql.DB and *sql.Tx used by handlers.
type DB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type createRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
}

type updateRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

func RegisterRoutes(r *gin.Engine, database DB) {
	g := r.Group("/users")
	g.GET("", list(database))
	g.POST("", create(database))
	g.GET("/:id", get(database))
	g.PUT("/:id", update(database))
	g.DELETE("/:id", delete(database))
}

func list(database DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `SELECT id, name, email, created_at, updated_at FROM users`
		var args []any

		if email, ok := c.GetQuery("email"); ok {
			query += ` WHERE email = $1`
			args = append(args, email)
		}

		query += ` ORDER BY created_at DESC`

		rows, err := database.QueryContext(c.Request.Context(), query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		defer func() { _ = rows.Close() }()

		result := []User{}
		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
			result = append(result, u)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func create(database DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var u User
		err := database.QueryRowContext(c.Request.Context(),
			`INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at, updated_at`,
			req.Name, req.Email,
		).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				c.JSON(http.StatusConflict, gin.H{"error": "email already in use"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, u)
	}
}

func get(database DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		var u User
		err = database.QueryRowContext(c.Request.Context(),
			`SELECT id, name, email, created_at, updated_at FROM users WHERE id = $1`, id,
		).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, u)
	}
}

func update(database DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		var req updateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var u User
		err = database.QueryRowContext(c.Request.Context(),
			`UPDATE users SET name = COALESCE($1, name), email = COALESCE($2, email), updated_at = NOW() WHERE id = $3 RETURNING id, name, email, created_at, updated_at`,
			req.Name, req.Email, id,
		).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, u)
	}
}

func delete(database DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		result, err := database.ExecContext(c.Request.Context(),
			`DELETE FROM users WHERE id = $1`, id)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				c.JSON(http.StatusConflict, gin.H{"error": "cannot delete user with existing todos"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if n, _ := result.RowsAffected(); n == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
