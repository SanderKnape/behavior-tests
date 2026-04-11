package users

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func userRouter(db DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, db)
	return r
}

var userColumns = []string{"id", "name", "email", "created_at", "updated_at"}

func TestList_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows(userColumns))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())
}

func TestList_ReturnsUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows(userColumns).
		AddRow(1, "Alice", "alice@example.com", now, now).
		AddRow(2, "Bob", "bob@example.com", now, now)
	mock.ExpectQuery(".*").WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "Alice", result[0].Name)
	assert.Equal(t, "Bob", result[1].Name)
}

func TestList_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(".*").WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(".*").
		WithArgs("Alice", "alice@example.com").
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow(1, "Alice", "alice@example.com", now, now))

	body := `{"name": "Alice", "email": "alice@example.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var result User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, "Alice", result.Name)
	assert.Equal(t, "alice@example.com", result.Email)
}

func TestCreate_BadRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(".*").WillReturnError(sql.ErrConnDone)

	body := `{"name": "Alice", "email": "alice@example.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGet_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(".*").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow(1, "Alice", "alice@example.com", now, now))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/1", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, int64(1), result.ID)
}

func TestGet_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows(userColumns))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/99", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGet_InvalidID(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/abc", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()
	name := "Alice Updated"
	email := "alice2@example.com"
	mock.ExpectQuery(".*").
		WithArgs(&name, &email, int64(1)).
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow(1, name, email, now, now))

	body := `{"name": "Alice Updated", "email": "alice2@example.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, name, result.Name)
	assert.Equal(t, email, result.Email)
}

func TestGet_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(".*").WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/1", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdate_InvalidID(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/abc", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_BadRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/1", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(".*").WillReturnError(sql.ErrConnDone)

	body := `{"name": "Ghost"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdate_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows(userColumns))

	body := `{"name": "Ghost"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/99", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(".*").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/users/1", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDelete_InvalidID(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/users/abc", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDelete_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(".*").WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/users/1", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDelete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(".*").
		WithArgs(int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/users/99", nil)
	userRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
