package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hmcts/task-manager/database"
	"github.com/hmcts/task-manager/handler"
	"github.com/hmcts/task-manager/model"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	h := handler.New(db)
	mux := http.NewServeMux()
	h.Routes(mux)
	return httptest.NewServer(mux)
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	return resp
}

func createTestTask(t *testing.T, base string) model.Task {
	t.Helper()
	resp := postJSON(t, base+"/api/tasks", map[string]any{
		"title":   "Test Task",
		"status":  "TODO",
		"dueDate": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	defer resp.Body.Close()
	var task model.Task
	_ = json.NewDecoder(resp.Body).Decode(&task)
	return task
}

func TestListTasks_Empty(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/tasks")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var tasks []model.Task
	_ = json.NewDecoder(resp.Body).Decode(&tasks)
	if len(tasks) != 0 {
		t.Errorf("expected empty list, got %d tasks", len(tasks))
	}
}

func TestCreateTask_Success(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/api/tasks", map[string]any{
		"title":       "My Task",
		"description": "A description",
		"status":      "TODO",
		"dueDate":     time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var task model.Task
	_ = json.NewDecoder(resp.Body).Decode(&task)
	if task.ID == "" {
		t.Error("expected task to have an ID")
	}
	if task.Title != "My Task" {
		t.Errorf("expected title 'My Task', got %q", task.Title)
	}
}

func TestCreateTask_MissingTitle(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/api/tasks", map[string]any{
		"status":  "TODO",
		"dueDate": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestCreateTask_InvalidStatus(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/api/tasks", map[string]any{
		"title":   "Task",
		"status":  "INVALID",
		"dueDate": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestGetTask_Success(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	created := createTestTask(t, srv.URL)

	resp, _ := http.Get(srv.URL + "/api/tasks/" + created.ID)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/tasks/nonexistent-id")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateStatus_Success(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	created := createTestTask(t, srv.URL)

	body, _ := json.Marshal(map[string]string{"status": "IN_PROGRESS"})
	req, _ := http.NewRequest(http.MethodPatch,
		srv.URL+"/api/tasks/"+created.ID+"/status",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var task model.Task
	_ = json.NewDecoder(resp.Body).Decode(&task)
	if task.Status != model.TaskStatusInProgress {
		t.Errorf("expected IN_PROGRESS, got %s", task.Status)
	}
}

func TestDeleteTask_Success(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	created := createTestTask(t, srv.URL)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/tasks/"+created.ID, nil)
	resp, _ := http.DefaultClient.Do(req)

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}

	// Confirm it's gone
	get, _ := http.Get(srv.URL + "/api/tasks/" + created.ID)
	if get.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", get.StatusCode)
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/tasks/nonexistent-id", nil)
	resp, _ := http.DefaultClient.Do(req)

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}