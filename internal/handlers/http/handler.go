package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"task-tracker/internal/models"
	postgresrepo "task-tracker/internal/repository/postgres"
	"task-tracker/web"
)

const maxRequestBody = 1 << 20

type Handler struct {
	repository *postgresrepo.Repository
	db         *pgxpool.Pool
	logger     *slog.Logger
}

func New(repo *postgresrepo.Repository, db *pgxpool.Pool, logger *slog.Logger) http.Handler {
	h := &Handler{repository: repo, db: db, logger: logger}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health/live", h.live)
	mux.HandleFunc("GET /health/ready", h.ready)

	mux.HandleFunc("GET /api/projects", h.listProjects)
	mux.HandleFunc("POST /api/projects", h.createProject)
	mux.HandleFunc("GET /api/projects/{id}", h.getProject)
	mux.HandleFunc("PUT /api/projects/{id}", h.updateProject)
	mux.HandleFunc("DELETE /api/projects/{id}", h.deleteProject)

	mux.HandleFunc("GET /api/tasks", h.listTasks)
	mux.HandleFunc("POST /api/tasks", h.createTask)
	mux.HandleFunc("GET /api/tasks/{id}", h.getTask)
	mux.HandleFunc("PUT /api/tasks/{id}", h.updateTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", h.deleteTask)

	mux.Handle("GET /", http.FileServer(http.FS(web.Files)))

	return h.recoverPanic(h.logRequest(mux))
}

func (h *Handler) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		h.logger.Error("readiness check failed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "database is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.repository.ListProjects(r.Context())
	if err != nil {
		h.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (h *Handler) getProject(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	project, err := h.repository.GetProject(r.Context(), id)
	if err != nil {
		h.repositoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	var input models.ProjectInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateProject(&input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	project, err := h.repository.CreateProject(r.Context(), input)
	if err != nil {
		h.internalError(w, err)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/projects/%d", project.ID))
	writeJSON(w, http.StatusCreated, project)
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	var input models.ProjectInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateProject(&input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	project, err := h.repository.UpdateProject(r.Context(), id, input)
	if err != nil {
		h.repositoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	if err := h.repository.DeleteProject(r.Context(), id); err != nil {
		h.repositoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	var projectID *models.ProjectID
	if value := r.URL.Query().Get("project_id"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "project_id must be a positive integer")
			return
		}
		value := models.ProjectID(parsed)
		projectID = &value
	}

	status := models.TaskStatus(r.URL.Query().Get("status"))
	if status != "" && !status.Valid() {
		writeError(w, http.StatusBadRequest, "unknown task status")
		return
	}

	tasks, err := h.repository.ListTasks(r.Context(), projectID, status)
	if err != nil {
		h.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	id, ok := taskID(w, r)
	if !ok {
		return
	}
	task, err := h.repository.GetTask(r.Context(), id)
	if err != nil {
		h.repositoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var input models.TaskInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateTask(&input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if _, err := h.repository.GetProject(r.Context(), input.ProjectID); err != nil {
		h.repositoryError(w, err)
		return
	}

	task, err := h.repository.CreateTask(r.Context(), input)
	if err != nil {
		h.internalError(w, err)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/tasks/%d", task.ID))
	writeJSON(w, http.StatusCreated, task)
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := taskID(w, r)
	if !ok {
		return
	}
	var input models.TaskInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateTask(&input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if _, err := h.repository.GetProject(r.Context(), input.ProjectID); err != nil {
		h.repositoryError(w, err)
		return
	}

	task, err := h.repository.UpdateTask(r.Context(), id, input)
	if err != nil {
		h.repositoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := taskID(w, r)
	if !ok {
		return
	}
	if err := h.repository.DeleteTask(r.Context(), id); err != nil {
		h.repositoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateProject(input *models.ProjectInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" {
		return errors.New("project name is required")
	}
	if len([]rune(input.Name)) > 120 {
		return errors.New("project name must not exceed 120 characters")
	}
	if len([]rune(input.Description)) > 5000 {
		return errors.New("project description must not exceed 5000 characters")
	}
	return nil
}

func validateTask(input *models.TaskInput) error {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.ProjectID < 1 {
		return errors.New("project_id must be a positive integer")
	}
	if input.Title == "" {
		return errors.New("task title is required")
	}
	if len([]rune(input.Title)) > 200 {
		return errors.New("task title must not exceed 200 characters")
	}
	if len([]rune(input.Description)) > 5000 {
		return errors.New("task description must not exceed 5000 characters")
	}
	if input.Status == "" {
		input.Status = models.TaskStatusTodo
	}
	if !input.Status.Valid() {
		return errors.New("status must be todo, in_progress, or done")
	}
	return nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

func projectID(w http.ResponseWriter, r *http.Request) (models.ProjectID, bool) {
	id, ok := pathID(w, r)
	return models.ProjectID(id), ok
}

func taskID(w http.ResponseWriter, r *http.Request) (models.TaskID, bool) {
	id, ok := pathID(w, r)
	return models.TaskID(id), ok
}

func (h *Handler) repositoryError(w http.ResponseWriter, err error) {
	if errors.Is(err, postgresrepo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "resource not found")
		return
	}
	h.internalError(w, err)
}

func (h *Handler) internalError(w http.ResponseWriter, err error) {
	h.logger.Error("request failed", "error", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (w *responseRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (h *Handler) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		h.logger.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

func (h *Handler) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Error("panic recovered", "error", recovered)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
