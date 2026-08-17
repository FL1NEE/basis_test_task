package httpserver

import (
	"net/http"
	"strconv"

	"github.com/FL1NEE/basis_test_task/internal/service"
)

type statsHandler struct {
	stats *service.StatsService
}

func (h *statsHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	teamID, err := strconv.ParseInt(r.PathValue("team_id"), 10, 64)
	if err != nil {
		writeError(w, badRequest("invalid team id"))
		return
	}

	stats, err := h.stats.GetTeamStats(r.Context(), userID, teamID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
