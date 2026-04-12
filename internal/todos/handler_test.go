package todos

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
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

func TestList_FilterCompleted(t *testing.T) {
	for _, completed := range []bool{true, false} {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() //nolint:errcheck

		now := time.Now()
		rows := sqlmock.NewRows(todoColumns).AddRow(1, 10, "Buy milk", completed, now, now)
		mock.ExpectQuery(".*").WithArgs(completed).WillReturnRows(rows)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/todos?completed="+strconv.FormatBool(completed), nil)
		todoRouter(db).ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result []Todo
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		assert.Len(t, result, 1)
		assert.Equal(t, completed, result[0].Completed)
	}
}

func TestList_FilterUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	rows := sqlmock.NewRows(todoColumns).AddRow(1, 42, "Buy milk", false, now, now)
	mock.ExpectQuery(".*").WithArgs(int64(42)).WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos?user_id=42", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, int64(42), result[0].UserID)
}

func TestList_FilterCompletedAndUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	rows := sqlmock.NewRows(todoColumns).AddRow(1, 42, "Buy milk", true, now, now)
	mock.ExpectQuery(".*").WithArgs(true, int64(42)).WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos?completed=true&user_id=42", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.True(t, result[0].Completed)
	assert.Equal(t, int64(42), result[0].UserID)
}

func TestList_FilterSearch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	rows := sqlmock.NewRows(todoColumns).AddRow(1, 10, "Buy milk", false, now, now)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, user_id, title, completed, created_at, updated_at FROM todos WHERE title ILIKE $1 ORDER BY created_at DESC, id DESC`,
	)).WithArgs("%milk%").WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos?search=milk", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "Buy milk", result[0].Title)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_Pagination_Limit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	rows := sqlmock.NewRows(todoColumns).AddRow(1, 10, "Buy milk", false, now, now)
	mock.ExpectQuery(`SELECT.*LIMIT \$1`).WithArgs(int64(5)).WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos?limit=5", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_Pagination_Offset(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	rows := sqlmock.NewRows(todoColumns).AddRow(2, 10, "Walk dog", true, now, now)
	mock.ExpectQuery(`SELECT.*OFFSET \$1`).WithArgs(int64(1)).WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos?offset=1", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_Pagination_LimitAndOffset(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	rows := sqlmock.NewRows(todoColumns).AddRow(2, 10, "Walk dog", true, now, now)
	mock.ExpectQuery(`SELECT.*LIMIT \$1.*OFFSET \$2`).WithArgs(int64(5), int64(10)).WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos?limit=5&offset=10", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_Pagination_InvalidLimit(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	for _, val := range []string{"abc", "0", "-1"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/todos?limit="+val, nil)
		todoRouter(db).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "expected 400 for limit=%s", val)
	}
}

func TestList_Pagination_InvalidOffset(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	for _, val := range []string{"abc", "-1"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/todos?offset="+val, nil)
		todoRouter(db).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "expected 400 for offset=%s", val)
	}
}

func TestList_FilterCompleted_InvalidValue(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos?completed=maybe", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestList_FilterUserID_InvalidValue(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/todos?user_id=abc", nil)
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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

func TestBulkComplete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	mock.ExpectQuery("UPDATE todos").
		WithArgs(int64(1), int64(2)).
		WillReturnRows(sqlmock.NewRows(todoColumns).
			AddRow(1, 10, "Buy milk", true, now, now).
			AddRow(2, 10, "Walk dog", true, now, now))

	body := `{"ids": [1, 2]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos/bulk-complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.True(t, result[0].Completed)
	assert.True(t, result[1].Completed)
}

func TestBulkComplete_EmptyIDs(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	body := `{"ids": []}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos/bulk-complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())
}

func TestBulkComplete_TooManyIDs(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	ids := make([]int64, maxBulkIDs+1)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	bodyBytes, _ := json.Marshal(map[string]any{"ids": ids})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos/bulk-complete", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkComplete_InvalidBody(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos/bulk-complete", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkComplete_MissingIDs(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos/bulk-complete", strings.NewReader(`{"ids": null}`))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkComplete_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery("UPDATE todos").WillReturnError(sql.ErrConnDone)

	body := `{"ids": [1, 2]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos/bulk-complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestBulkComplete_NoneMatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery("UPDATE todos").
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows(todoColumns))

	body := `{"ids": [999]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos/bulk-complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())
}

func TestBulkComplete_RowsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	now := time.Now()
	mock.ExpectQuery("UPDATE todos").WillReturnRows(
		sqlmock.NewRows(todoColumns).
			AddRow(1, 10, "title", true, now, now).
			RowError(0, sql.ErrConnDone),
	)

	body := `{"ids": [1]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/todos/bulk-complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	todoRouter(db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
