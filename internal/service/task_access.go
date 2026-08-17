package service

import "github.com/FL1NEE/basis_test_task/internal/domain"

type taskAccessLevel int

const (
	taskAccessNone taskAccessLevel = iota
	taskAccessStatusOnly
	taskAccessFull
)

// taskEditAccess decides how much of a task a given team member may
// change, per the role rules from the spec:
//   - owner/admin: any task in the team, every field.
//   - the task's creator: every field, but only on their own task.
//   - the task's assignee, when not also creator/owner/admin: status only,
//     and they may never reassign the task to someone else.
//   - anyone else: nothing.
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
