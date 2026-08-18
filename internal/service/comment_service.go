package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/FL1NEE/basis_test_task/internal/repository"
)

type CommentService struct {
	comments *repository.CommentRepo
	tasks    *repository.TaskRepo
	teamSvc  *TeamService
}

func NewCommentService(comments *repository.CommentRepo, tasks *repository.TaskRepo, teamSvc *TeamService) *CommentService {
	return &CommentService{comments: comments, tasks: tasks, teamSvc: teamSvc}
}

// AddComment allows any team member, not just the task's assignee.
func (s *CommentService) AddComment(ctx context.Context, actingUserID, taskID int64, content string) (*domain.TaskComment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: comment content is required", domain.ErrValidation)
	}

	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := s.teamSvc.RequireMembership(ctx, task.TeamID, actingUserID); err != nil {
		return nil, err
	}

	id, err := s.comments.Create(ctx, taskID, actingUserID, content)
	if err != nil {
		return nil, err
	}

	return &domain.TaskComment{ID: id, TaskID: taskID, UserID: actingUserID, Content: content}, nil
}

func (s *CommentService) ListComments(ctx context.Context, actingUserID, taskID int64) ([]domain.TaskComment, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := s.teamSvc.RequireMembership(ctx, task.TeamID, actingUserID); err != nil {
		return nil, err
	}

	return s.comments.ListByTask(ctx, taskID)
}
