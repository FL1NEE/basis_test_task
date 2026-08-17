package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/FL1NEE/basis_test_task/internal/service"
)

type teamHandler struct {
	teams *service.TeamService
}

type createTeamRequest struct {
	Name string `json:"name"`
}

type createTeamResponse struct {
	TeamID int64 `json:"team_id"`
}

func (h *teamHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, badRequest("invalid request body"))
		return
	}

	id, err := h.teams.CreateTeam(r.Context(), userID, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, createTeamResponse{TeamID: id})
}

func (h *teamHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	teams, err := h.teams.ListMyTeams(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, teams)
}

type inviteRequest struct {
	Email string      `json:"email"`
	Role  domain.Role `json:"role"`
}

func (h *teamHandler) invite(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromContext(r.Context())

	teamID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, badRequest("invalid team id"))
		return
	}

	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, badRequest("invalid request body"))
		return
	}
	if req.Role == "" {
		req.Role = domain.RoleMember
	}

	if err := h.teams.InviteMember(r.Context(), userID, teamID, req.Email, req.Role); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
