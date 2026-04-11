//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"me/internal/platform/db"
	"me/internal/todos"
	"me/internal/users"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:18.3",
		tcpostgres.WithDatabase("todos"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Printf("failed to start postgres container: %v\n", err)
		return 1
	}
	defer pgContainer.Terminate(ctx) //nolint:errcheck

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Printf("failed to get connection string: %v\n", err)
		return 1
	}

	os.Setenv("DATABASE_URL", connStr)

	var dbErr error
	testDB, dbErr = db.Open()
	if dbErr != nil {
		fmt.Printf("failed to open database: %v\n", dbErr)
		return 1
	}
	defer testDB.Close() //nolint:errcheck

	if err := db.RunMigrations(testDB); err != nil {
		fmt.Printf("failed to run migrations: %v\n", err)
		return 1
	}
	if err := db.RunSeeds(testDB); err != nil {
		fmt.Printf("failed to run seeds: %v\n", err)
		return 1
	}

	gin.SetMode(gin.TestMode)

	return m.Run()
}

// testEnv holds a per-test router backed by a REPEATABLE READ transaction.
// The transaction is rolled back automatically via t.Cleanup, so each test
// starts from a clean slate and tests can safely run in parallel.
type testEnv struct {
	router *gin.Engine
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	tx, err := testDB.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() { tx.Rollback() }) //nolint:errcheck

	return &testEnv{router: setupRouter(tx)}
}

func (e *testEnv) doRequest(method, path string, body any) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

// createTodo is a helper that POSTs a new todo, asserts the 201 response, and
// returns the decoded todo. Use this for test setup rather than inline calls.
func createTodo(t *testing.T, env *testEnv, title string, userID int64) todos.Todo {
	t.Helper()

	w := env.doRequest(http.MethodPost, "/todos", map[string]any{
		"title":   title,
		"user_id": userID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("setup POST /todos: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	return decode[todos.Todo](w)
}

// createUser is a helper that POSTs a new user, asserts the 201 response, and
// returns the decoded user. Use this for test setup rather than inline calls.
func createUser(t *testing.T, env *testEnv, name, email string) users.User {
	t.Helper()
	w := env.doRequest(http.MethodPost, "/users", map[string]any{
		"name":  name,
		"email": email,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("setup POST /users: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	return decode[users.User](w)
}

// decode unmarshals the JSON response body into T.
func decode[T any](w *httptest.ResponseRecorder) T {
	var v T
	json.NewDecoder(w.Body).Decode(&v) //nolint:errcheck
	return v
}
