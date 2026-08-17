package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/FL1NEE/basis_test_task/internal/domain"
)

func badRequest(message string) error {
	return fmt.Errorf("%w: %s", domain.ErrValidation, message)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode response", "error", err)
	}
}

type errorBody struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, err error) {
	status, message := mapError(err)
	if status == http.StatusInternalServerError {
		slog.Error("unhandled error", "error", err)
		message = "internal server error"
	}
	writeJSON(w, status, errorBody{Error: message})
}

func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "missing or invalid authorization token"
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not found"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, domain.ErrNotTeamMember):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, domain.ErrEmailTaken):
		return http.StatusConflict, "email already registered"
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, "invalid email or password"
	case errors.Is(err, domain.ErrVersionMismatch):
		return http.StatusConflict, "task was modified by someone else, refetch and retry"
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
