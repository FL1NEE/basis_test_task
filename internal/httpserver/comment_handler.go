package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FL1NEE/basis_test_task/internal/service"
)

type commentHandler struct {
	comments *service.CommentService
}

type createCommentRequest struct {
	Content string `json:"content"`
}

func (h *commentHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, badRequest("invalid task id"))
		return
	}

	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, badRequest("invalid request body"))
		return
	}

	comment, err := h.comments.AddComment(r.Context(), userID, taskID, req.Content)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, comment)
}

func (h *commentHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, badRequest("invalid task id"))
		return
	}

	comments, err := h.comments.ListComments(r.Context(), userID, taskID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, comments)
}
