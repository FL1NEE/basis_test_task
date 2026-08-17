package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/FL1NEE/basis_test_task/internal/service"
)

type taskHandler struct {
	tasks *service.TaskService
}

type createTaskRequest struct {
	TeamID      int64   `json:"team_id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	AssigneeID  *int64  `json:"assignee_id"`
}

func (h *taskHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, badRequest("invalid request body"))
		return
	}

	task, err := h.tasks.CreateTask(r.Context(), userID, req.TeamID, req.Title, req.Description, req.AssigneeID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *taskHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())
	q := r.URL.Query()

	teamID, err := strconv.ParseInt(q.Get("team_id"), 10, 64)
	if err != nil {
		writeError(w, badRequest("team_id is required"))
		return
	}

	params := service.ListTasksParams{TeamID: teamID}

	if statusStr := q.Get("status"); statusStr != "" {
		status := domain.TaskStatus(statusStr)
		params.Status = &status
	}
	if assigneeStr := q.Get("assignee_id"); assigneeStr != "" {
		assigneeID, err := strconv.ParseInt(assigneeStr, 10, 64)
		if err != nil {
			writeError(w, badRequest("invalid assignee_id"))
			return
		}
		params.AssigneeID = &assigneeID
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			writeError(w, badRequest("invalid limit"))
			return
		}
		params.Limit = limit
	}
	if offsetStr := q.Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			writeError(w, badRequest("invalid offset"))
			return
		}
		params.Offset = offset
	}

	tasks, err := h.tasks.ListTasks(r.Context(), userID, params)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

type updateTaskRequest struct {
	Version     int                `json:"version"`
	Title       *string            `json:"title"`
	Description *string            `json:"description"`
	Status      *domain.TaskStatus `json:"status"`
	AssigneeID  *int64             `json:"assignee_id"`
}

func (h *taskHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, badRequest("invalid task id"))
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, badRequest("invalid request body"))
		return
	}

	task, err := h.tasks.UpdateTask(r.Context(), userID, taskID, req.Version, service.TaskPatch{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}
