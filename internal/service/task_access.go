package service

import "github.com/FL1NEE/basis_test_task/internal/domain"

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
