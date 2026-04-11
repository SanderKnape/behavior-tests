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

func TestBehavior_Todo_Create_Returns422ForNonExistentUserID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPost, "/todos", map[string]any{"title": "orphan todo", "user_id": 999999})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBehavior_Todo_List_FilterByCompleted_ReturnsOnlyCompletedTodos(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	created := createTodo(t, env, "complete me", 1)
	env.doRequest(http.MethodPut, fmt.Sprintf("/todos/%d", created.ID), map[string]any{"completed": true})

	createTodo(t, env, "leave me incomplete", 1)

	w := env.doRequest(http.MethodGet, "/todos?completed=true", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := decode[[]todos.Todo](w)
	for _, todo := range result {
		if !todo.Completed {
			t.Fatalf("expected only completed todos, got todo %d with completed=false", todo.ID)
		}
	}
}

func TestBehavior_Todo_List_FilterByCompleted_ReturnsOnlyIncompleteTodos(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	created := createTodo(t, env, "complete me", 1)
	env.doRequest(http.MethodPut, fmt.Sprintf("/todos/%d", created.ID), map[string]any{"completed": true})

	createTodo(t, env, "leave me incomplete", 1)

	w := env.doRequest(http.MethodGet, "/todos?completed=false", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := decode[[]todos.Todo](w)
	for _, todo := range result {
		if todo.Completed {
			t.Fatalf("expected only incomplete todos, got todo %d with completed=true", todo.ID)
		}
	}
}

func TestBehavior_Todo_List_FilterByCompleted_Returns400ForInvalidValue(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodGet, "/todos?completed=maybe", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBehavior_Todo_List_FilterByUserID_ReturnsOnlyMatchingUsersTodos(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	user := createUser(t, env, "Todo Owner", "todo-owner@example.com")
	otherUser := createUser(t, env, "Other Owner", "other-owner@example.com")

	matching := createTodo(t, env, "match me", user.ID)
	other := createTodo(t, env, "not me", otherUser.ID)

	w := env.doRequest(http.MethodGet, fmt.Sprintf("/todos?user_id=%d", user.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := decode[[]todos.Todo](w)
	foundMatching := false
	for _, todo := range result {
		if todo.ID == other.ID {
			t.Fatalf("expected todo %d to be excluded from user_id filter", other.ID)
		}
		if todo.UserID != user.ID {
			t.Fatalf("expected only todos for user %d, got todo %d for user %d", user.ID, todo.ID, todo.UserID)
		}
		if todo.ID == matching.ID {
			foundMatching = true
		}
	}

	if !foundMatching {
		t.Fatalf("expected filtered list to include created todo %d", matching.ID)
	}
}

func TestBehavior_Todo_List_FilterByCompletedAndUserID_ReturnsOnlyMatchingTodos(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	user := createUser(t, env, "Combined Filter User", "combined-filter@example.com")
	otherUser := createUser(t, env, "Other Combined User", "other-combined@example.com")

	matching := createTodo(t, env, "match me", user.ID)
	env.doRequest(http.MethodPut, fmt.Sprintf("/todos/%d", matching.ID), map[string]any{"completed": true})

	sameUserIncomplete := createTodo(t, env, "same user incomplete", user.ID)
	otherUserCompleted := createTodo(t, env, "other user complete", otherUser.ID)
	env.doRequest(http.MethodPut, fmt.Sprintf("/todos/%d", otherUserCompleted.ID), map[string]any{"completed": true})

	w := env.doRequest(http.MethodGet, fmt.Sprintf("/todos?completed=true&user_id=%d", user.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := decode[[]todos.Todo](w)
	foundMatching := false
	for _, todo := range result {
		if todo.ID == sameUserIncomplete.ID {
			t.Fatalf("expected incomplete todo %d to be excluded from combined filter", sameUserIncomplete.ID)
		}
		if todo.ID == otherUserCompleted.ID {
			t.Fatalf("expected other user's todo %d to be excluded from combined filter", otherUserCompleted.ID)
		}
		if todo.UserID != user.ID {
			t.Fatalf("expected only todos for user %d, got todo %d for user %d", user.ID, todo.ID, todo.UserID)
		}
		if !todo.Completed {
			t.Fatalf("expected only completed todos, got todo %d with completed=false", todo.ID)
		}
		if todo.ID == matching.ID {
			foundMatching = true
		}
	}

	if !foundMatching {
		t.Fatalf("expected filtered list to include created todo %d", matching.ID)
	}
}

func TestBehavior_Todo_List_FilterByUserID_Returns400ForInvalidValue(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodGet, "/todos?user_id=abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
