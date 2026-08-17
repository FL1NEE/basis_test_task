package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/FL1NEE/basis_test_task/internal/service"
)

type historyHandler struct {
	history *service.HistoryService
}

// taskHistoryEntryResponse mirrors domain.TaskHistory but exposes Changes
// as json.RawMessage: the repository stores/returns it as a plain string
// of raw JSON, and marshaling a Go string field verbatim would double-encode
// it into a JSON string (`"{\"status\":...}"`) instead of a nested object
// (`{"status":...}`), which is what the OpenAPI schema promises callers.
type taskHistoryEntryResponse struct {
	ID        int64           `json:"id"`
	TaskID    int64           `json:"task_id"`
	ChangedBy int64           `json:"changed_by"`
	Changes   json.RawMessage `json:"changes"`
	CreatedAt time.Time       `json:"created_at"`
}

func (h *historyHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, badRequest("invalid task id"))
		return
	}

	history, err := h.history.ListHistory(r.Context(), userID, taskID)
	if err != nil {
		writeError(w, err)
		return
	}

	response := make([]taskHistoryEntryResponse, len(history))
	for i, entry := range history {
		response[i] = taskHistoryEntryResponse{
			ID:        entry.ID,
			TaskID:    entry.TaskID,
			ChangedBy: entry.ChangedBy,
			Changes:   json.RawMessage(entry.Changes),
			CreatedAt: entry.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, response)
}
