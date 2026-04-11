package todos

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

type Todo struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type createRequest struct {
	UserID int64  `json:"user_id" binding:"required"`
	Title  string `json:"title" binding:"required"`
}

type updateRequest struct {
	Title     *string `json:"title"`
	Completed *bool   `json:"completed"`
}

func RegisterRoutes(r *gin.Engine, database DB) {
	g := r.Group("/todos")
	g.GET("", list(database))
	g.POST("", create(database))
	g.GET("/:id", get(database))
	g.PUT("/:id", update(database))
	g.DELETE("/:id", delete(database))
}

func list(database DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `SELECT id, user_id, title, completed, created_at, updated_at FROM todos`
		var args []any

		if raw, ok := c.GetQuery("completed"); ok {
			completed, err := strconv.ParseBool(raw)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "completed must be true or false"})
				return
			}
			query += ` WHERE completed = $1`
			args = append(args, completed)
		}

		query += ` ORDER BY created_at DESC`

		rows, err := database.QueryContext(c.Request.Context(), query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		defer func() { _ = rows.Close() }()

		result := []Todo{}
		for rows.Next() {
			var t Todo
			if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
			result = append(result, t)
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

		var t Todo
		err := database.QueryRowContext(c.Request.Context(),
			`INSERT INTO todos (user_id, title) VALUES ($1, $2)
			 RETURNING id, user_id, title, completed, created_at, updated_at`,
			req.UserID, req.Title,
		).Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "referenced user does not exist"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, t)
	}
}

func get(database DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		var t Todo
		err = database.QueryRowContext(c.Request.Context(),
			`SELECT id, user_id, title, completed, created_at, updated_at FROM todos WHERE id = $1`, id,
		).Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, t)
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

		var t Todo
		err = database.QueryRowContext(c.Request.Context(),
			`UPDATE todos
			 SET title      = COALESCE($1, title),
			     completed  = COALESCE($2, completed),
			     updated_at = NOW()
			 WHERE id = $3
			 RETURNING id, user_id, title, completed, created_at, updated_at`,
			req.Title, req.Completed, id,
		).Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, t)
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
			`DELETE FROM todos WHERE id = $1`, id)
		if err != nil {
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
