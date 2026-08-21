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
	tasks    *TaskService
}

func NewCommentService(comments *repository.CommentRepo, tasks *TaskService) *CommentService {
	return &CommentService{comments: comments, tasks: tasks}
}

// AddComment allows any team member, not just the task's assignee.
func (s *CommentService) AddComment(ctx context.Context, actingUserID, taskID int64, content string) (*domain.TaskComment, error) {
	if _, _, err := s.tasks.GetVisibleTask(ctx, actingUserID, taskID); err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: comment content is required", domain.ErrValidation)
	}

	id, err := s.comments.Create(ctx, taskID, actingUserID, content)
	if err != nil {
		return nil, err
	}

	return &domain.TaskComment{ID: id, TaskID: taskID, UserID: actingUserID, Content: content}, nil
}

func (s *CommentService) ListComments(ctx context.Context, actingUserID, taskID int64) ([]domain.TaskComment, error) {
	if _, _, err := s.tasks.GetVisibleTask(ctx, actingUserID, taskID); err != nil {
		return nil, err
	}

	return s.comments.ListByTask(ctx, taskID)
}
