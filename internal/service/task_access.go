package service

import (
	"context"
	"errors"

	"github.com/FL1NEE/basis_test_task/internal/domain"
)

// requireTaskTeamMembership checks membership in a task's team and, on
// failure, reports it as domain.ErrNotFound rather than domain.ErrForbidden.
// A caller who isn't on the task's team must not be able to tell "this
// task doesn't exist" apart from "it exists, but isn't yours" - both cases
// have to look identical from the outside, or task IDs become enumerable
// across teams by diffing 404 vs 403 responses.
func requireTaskTeamMembership(teamSvc *TeamService, ctx context.Context, teamID, userID int64) (domain.Role, error) {
	role, err := teamSvc.RequireMembership(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return "", domain.ErrNotFound
		}
		return "", err
	}
	return role, nil
}

type taskAccessLevel int

const (
	taskAccessNone taskAccessLevel = iota
	taskAccessStatusOnly
	taskAccessFull
)

// owner/admin or the task's creator get full access; the assignee gets
// status-only; everyone else gets none.
func taskEditAccess(role domain.Role, task *domain.Task, userID int64) taskAccessLevel {
	if role == domain.RoleOwner || role == domain.RoleAdmin {
		return taskAccessFull
	}
	if task.CreatedBy == userID {
		return taskAccessFull
	}
	if task.AssigneeID != nil && *task.AssigneeID == userID {
		return taskAccessStatusOnly
	}
	return taskAccessNone
}
