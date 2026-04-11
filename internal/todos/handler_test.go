package todos

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

func todoRouter(db DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, db)
	return r
}

var todoColumns = []string{"id", "user_id", "title", "completed", "created_at", "updated_at"}

func TestList_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows(todoColumns))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())
}

func TestList_ReturnsTodos(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	rows := sqlmock.NewRows(todoColumns).
		AddRow(1, 10, "Buy milk", false, now, now).
		AddRow(2, 10, "Walk dog", true, now, now)
	mock.ExpectQuery(".*").WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "Buy milk", result[0].Title)
	assert.Equal(t, "Walk dog", result[1].Title)
}

func TestList_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(".*").WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	mock.ExpectQuery(".*").
		WithArgs(int64(10), "Buy milk").
		WillReturnRows(sqlmock.NewRows(todoColumns).AddRow(1, 10, "Buy milk", false, now, now))

	body := `{"user_id": 10, "title": "Buy milk"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var result Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, "Buy milk", result.Title)
	assert.Equal(t, int64(10), result.UserID)
}

func TestCreate_BadRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(".*").WillReturnError(sql.ErrConnDone)

	body := `{"user_id": 10, "title": "Buy milk"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGet_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	mock.ExpectQuery(".*").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows(todoColumns).AddRow(1, 10, "Buy milk", false, now, now))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos/1", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, int64(1), result.ID)
}

func TestGet_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows(todoColumns))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos/99", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGet_InvalidID(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos/abc", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	title := "Updated title"
	completed := true
	mock.ExpectQuery(".*").
		WithArgs(&title, &completed, int64(1)).
		WillReturnRows(sqlmock.NewRows(todoColumns).AddRow(1, 10, title, completed, now, now))

	body := `{"title": "Updated title", "completed": true}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/todos/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, title, result.Title)
	assert.True(t, result.Completed)
}

func TestGet_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(".*").WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos/1", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdate_InvalidID(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/todos/abc", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_BadRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/todos/1", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(".*").WillReturnError(sql.ErrConnDone)

	body := `{"title": "Updated"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/todos/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdate_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows(todoColumns))

	body := `{"title": "Updated"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/todos/99", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectExec(".*").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/todos/1", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDelete_InvalidID(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/todos/abc", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDelete_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectExec(".*").WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/todos/1", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestList_RowsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	mock.ExpectQuery(".*").WillReturnRows(
		sqlmock.NewRows(todoColumns).
			AddRow(1, 10, "title", false, now, now).
			RowError(0, sql.ErrConnDone),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDelete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectExec(".*").
		WithArgs(int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/todos/99", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
