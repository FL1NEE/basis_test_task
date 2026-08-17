package service

import (
	"testing"

	"github.com/FL1NEE/basis_test_task/internal/domain"
)

func TestTaskEditAccess(t *testing.T) {
	const (
		ownerID    int64 = 1
		adminID    int64 = 2
		creatorID  int64 = 3
		assigneeID int64 = 4
		outsiderID int64 = 5
	)

	task := &domain.Task{
		CreatedBy:  creatorID,
		AssigneeID: ptr(assigneeID),
	}

	cases := []struct {
		name   string
		role   domain.Role
		userID int64
		want   taskAccessLevel
	}{
		{"owner can edit any task", domain.RoleOwner, ownerID, taskAccessFull},
		{"admin can edit any task", domain.RoleAdmin, adminID, taskAccessFull},
		{"creator can edit own task", domain.RoleMember, creatorID, taskAccessFull},
		{"assignee can only change status", domain.RoleMember, assigneeID, taskAccessStatusOnly},
		{"unrelated member has no access", domain.RoleMember, outsiderID, taskAccessNone},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := taskEditAccess(tc.role, task, tc.userID)
			if got != tc.want {
				t.Errorf("taskEditAccess(%v, task, %d) = %v, want %v", tc.role, tc.userID, got, tc.want)
			}
		})
	}
}

func TestTaskEditAccess_UnassignedTask(t *testing.T) {
	task := &domain.Task{CreatedBy: 1, AssigneeID: nil}

	got := taskEditAccess(domain.RoleMember, task, 2)
	if got != taskAccessNone {
		t.Errorf("expected no access for a member who is neither creator nor assignee, got %v", got)
	}
}

func TestTaskEditAccess_CreatorWhoIsAlsoAssignee(t *testing.T) {
	task := &domain.Task{CreatedBy: 1, AssigneeID: ptr[int64](1)}

	got := taskEditAccess(domain.RoleMember, task, 1)
	if got != taskAccessFull {
		t.Errorf("creator should get full access even when also assigned, got %v", got)
	}
}

func ptr[T any](v T) *T {
	return &v
}
