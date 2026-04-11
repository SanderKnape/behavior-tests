//go:build integration

package behavior

import (
	"fmt"
	"net/http"
	"testing"

	"me/internal/users"
)

func TestBehavior_User_List_ReturnsSeededUsers(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodGet, "/users", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := decode[[]users.User](w)
	if len(result) == 0 {
		t.Fatal("expected seeded users, got empty list")
	}
}

func TestBehavior_User_Create_PersistsAndReturns(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPost, "/users", map[string]any{
		"name":  "Dave Test",
		"email": "dave@example.com",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	user := decode[users.User](w)
	if user.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if user.Name != "Dave Test" {
		t.Fatalf("unexpected name: %s", user.Name)
	}
	if user.Email != "dave@example.com" {
		t.Fatalf("unexpected email: %s", user.Email)
	}
}

func TestBehavior_User_Create_RejectsMissingName(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPost, "/users", map[string]any{"email": "noname@example.com"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBehavior_User_Create_RejectsMissingEmail(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPost, "/users", map[string]any{"name": "No Email"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBehavior_User_Get_ReturnsMatchingUser(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	setup := env.doRequest(http.MethodPost, "/users", map[string]any{
		"name":  "Get Me",
		"email": "getme@example.com",
	})
	if setup.Code != http.StatusCreated {
		t.Fatalf("setup POST /users: expected 201, got %d: %s", setup.Code, setup.Body.String())
	}
	created := decode[users.User](setup)

	w := env.doRequest(http.MethodGet, fmt.Sprintf("/users/%d", created.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	user := decode[users.User](w)
	if user.ID != created.ID {
		t.Fatalf("expected ID %d, got %d", created.ID, user.ID)
	}
}

func TestBehavior_User_Get_Returns404ForUnknownID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodGet, "/users/999999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestBehavior_User_Update_PersistsChanges(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	setup := env.doRequest(http.MethodPost, "/users", map[string]any{
		"name":  "Before",
		"email": "before@example.com",
	})
	if setup.Code != http.StatusCreated {
		t.Fatalf("setup POST /users: expected 201, got %d: %s", setup.Code, setup.Body.String())
	}
	created := decode[users.User](setup)

	w := env.doRequest(http.MethodPut, fmt.Sprintf("/users/%d", created.ID), map[string]any{
		"name":  "After",
		"email": "after@example.com",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	user := decode[users.User](w)
	if user.Name != "After" {
		t.Fatalf("expected name 'After', got %q", user.Name)
	}
	if user.Email != "after@example.com" {
		t.Fatalf("expected email 'after@example.com', got %q", user.Email)
	}
}

func TestBehavior_User_Update_Returns404ForUnknownID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPut, "/users/999999", map[string]any{"name": "Ghost"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestBehavior_User_Delete_RemovesAndReturns204(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	setup := env.doRequest(http.MethodPost, "/users", map[string]any{
		"name":  "Delete Me",
		"email": "deleteme@example.com",
	})
	if setup.Code != http.StatusCreated {
		t.Fatalf("setup POST /users: expected 201, got %d: %s", setup.Code, setup.Body.String())
	}
	created := decode[users.User](setup)

	w := env.doRequest(http.MethodDelete, fmt.Sprintf("/users/%d", created.ID), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	w = env.doRequest(http.MethodGet, fmt.Sprintf("/users/%d", created.ID), nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}
}

func TestBehavior_User_Delete_Returns404ForUnknownID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodDelete, "/users/999999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestBehavior_User_Get_Returns400ForInvalidID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodGet, "/users/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBehavior_User_Update_Returns400ForInvalidID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodPut, "/users/abc", map[string]any{"name": "x"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBehavior_User_Delete_Returns400ForInvalidID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodDelete, "/users/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBehavior_User_Create_Returns409ForDuplicateEmail(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	createUser(t, env, "Original User", "duplicate@example.com")

	w := env.doRequest(http.MethodPost, "/users", map[string]any{
		"name":  "Duplicate User",
		"email": "duplicate@example.com",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBehavior_User_Delete_Returns409WhenUserHasTodos(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	user := createUser(t, env, "User With Todos", "userwithTodos@example.com")
	createTodo(t, env, "blocked todo", user.ID)

	w := env.doRequest(http.MethodDelete, fmt.Sprintf("/users/%d", user.ID), nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBehavior_User_List_FilterByEmail_ReturnsEmptyListForNonMatchingEmail(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	w := env.doRequest(http.MethodGet, "/users?email=does-not-exist@example.com", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := decode[[]users.User](w)
	if len(result) != 0 {
		t.Fatalf("expected empty list for non-matching email filter, got %d users", len(result))
	}
}

func TestBehavior_User_List_FilterByEmail_ReturnsOnlyMatchingUsers(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	matching := createUser(t, env, "Email Filter User", "email-filter@example.com")
	other := createUser(t, env, "Other User", "other-user@example.com")

	w := env.doRequest(http.MethodGet, "/users?email=email-filter@example.com", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := decode[[]users.User](w)
	foundMatching := false
	for _, user := range result {
		if user.ID == other.ID {
			t.Fatalf("expected user %d to be excluded from email filter", other.ID)
		}
		if user.Email != matching.Email {
			t.Fatalf("expected only users with email %q, got user %d with email %q", matching.Email, user.ID, user.Email)
		}
		if user.ID == matching.ID {
			foundMatching = true
		}
	}

	if !foundMatching {
		t.Fatalf("expected filtered list to include created user %d", matching.ID)
	}
}
