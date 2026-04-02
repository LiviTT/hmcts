package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/hmcts/server/internal/database"
	"github.com/hmcts/server/internal/model"
)

type Handler struct {
	db *database.DB
}

func New(db *database.DB) *Handler {
	return &Handler{db: db}
}

// Routes registers all task routes onto the given mux.
//
//	GET    /api/tasks
//	POST   /api/tasks
//	GET    /api/tasks/{id}
//	PATCH  /api/tasks/{id}/status
//	DELETE /api/tasks/{id}
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.listTasks(w, r)
		case http.MethodPost:
			h.createTask(w, r)
		default:
			methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		// Strip "/api/tasks/" prefix and split remaining path
		path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
		parts := strings.Split(strings.Trim(path, "/"), "/")

		id := parts[0]
		if id == "" {
			writeError(w, http.StatusBadRequest, "missing task id")
			return
		}

		// /api/tasks/{id}/status
		if len(parts) == 2 && parts[1] == "status" {
			if r.Method == http.MethodPatch {
				h.updateStatus(w, r, id)
			} else {
				methodNotAllowed(w)
			}
			return
		}

		// /api/tasks/{id}
		switch r.Method {
		case http.MethodGet:
			h.getTask(w, r, id)
		case http.MethodDelete:
			h.deleteTask(w, r, id)
		default:
			methodNotAllowed(w)
		}
	})
}

// GET /api/tasks
func (h *Handler) listTasks(w http.ResponseWriter, _ *http.Request) {
	tasks, err := h.db.GetAllTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve tasks")
		return
	}
	if tasks == nil {
		tasks = []*model.Task{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

// POST /api/tasks
func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateCreateRequest(req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	task, err := h.db.CreateTask(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

// GET /api/tasks/{id}
func (h *Handler) getTask(w http.ResponseWriter, _ *http.Request, id string) {
	task, err := h.db.GetTaskByID(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to retrieve task")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// PATCH /api/tasks/{id}/status
func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request, id string) {
	var req model.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !req.Status.IsValid() {
		writeError(w, http.StatusUnprocessableEntity, "status must be TODO, IN_PROGRESS, or DONE")
		return
	}

	task, err := h.db.UpdateTaskStatus(id, req.Status)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// DELETE /api/tasks/{id}
func (h *Handler) deleteTask(w http.ResponseWriter, _ *http.Request, id string) {
	if err := h.db.DeleteTask(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Validation ---

func validateCreateRequest(req model.CreateTaskRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return errors.New("title is required")
	}
	if len(req.Title) > 200 {
		return errors.New("title must be 200 characters or fewer")
	}
	if !req.Status.IsValid() {
		return errors.New("status must be TODO, IN_PROGRESS, or DONE")
	}
	if req.DueDate.IsZero() {
		return errors.New("dueDate is required")
	}
	return nil
}

// --- Helpers ---

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}