//go:build integration

package behavior

import (
	"fmt"
	"net/http"
	"testing"

	"me/internal/todos"
)

func TestBehavior_Todo_List_ReturnsSeededTodos(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodGet, "/todos", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := decode[[]todos.Todo](w)
	if len(result) == 0 {
		t.Fatal("expected seeded todos, got empty list")
	}
}

func TestBehavior_Todo_Create_PersistsAndReturns(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPost, "/todos", map[string]any{"title": "integration test todo", "user_id": 1})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	todo := decode[todos.Todo](w)
	if todo.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if todo.Title != "integration test todo" {
		t.Fatalf("unexpected title: %s", todo.Title)
	}
	if todo.Completed {
		t.Fatal("new todo should not be completed")
	}
	if todo.UserID != 1 {
		t.Fatalf("expected user_id 1, got %d", todo.UserID)
	}
}

func TestBehavior_Todo_Create_RejectsMissingTitle(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPost, "/todos", map[string]any{"user_id": 1})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBehavior_Todo_Create_RejectsMissingUserID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPost, "/todos", map[string]any{"title": "no user"})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBehavior_Todo_Get_ReturnsMatchingTodo(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	created := createTodo(t, env, "get me", 1)

	w := env.doRequest(http.MethodGet, fmt.Sprintf("/todos/%d", created.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	todo := decode[todos.Todo](w)
	if todo.ID != created.ID {
		t.Fatalf("expected ID %d, got %d", created.ID, todo.ID)
	}
}

func TestBehavior_Todo_Get_Returns404ForUnknownID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodGet, "/todos/999999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestBehavior_Todo_Update_PersistsChanges(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	created := createTodo(t, env, "before", 1)

	w := env.doRequest(http.MethodPut, fmt.Sprintf("/todos/%d", created.ID), map[string]any{
		"title":     "after",
		"completed": true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	todo := decode[todos.Todo](w)
	if todo.Title != "after" {
		t.Fatalf("expected title 'after', got %q", todo.Title)
	}
	if !todo.Completed {
		t.Fatal("expected completed=true")
	}
}

func TestBehavior_Todo_Update_Returns404ForUnknownID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPut, "/todos/999999", map[string]any{"title": "ghost"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestBehavior_Todo_Delete_RemovesAndReturns204(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	created := createTodo(t, env, "delete me", 1)

	w := env.doRequest(http.MethodDelete, fmt.Sprintf("/todos/%d", created.ID), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	w = env.doRequest(http.MethodGet, fmt.Sprintf("/todos/%d", created.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}
}

func TestBehavior_Todo_Delete_Returns404ForUnknownID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodDelete, "/todos/999999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestBehavior_Todo_Get_Returns400ForInvalidID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodGet, "/todos/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBehavior_Todo_Update_Returns400ForInvalidID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPut, "/todos/abc", map[string]any{"title": "x"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBehavior_Todo_Delete_Returns400ForInvalidID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodDelete, "/todos/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
