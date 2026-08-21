package service

import (
	"context"

	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/FL1NEE/basis_test_task/internal/repository"
)

type HistoryService struct {
	history *repository.HistoryRepo
	tasks   *TaskService
}

func NewHistoryService(history *repository.HistoryRepo, tasks *TaskService) *HistoryService {
	return &HistoryService{history: history, tasks: tasks}
}

func (s *HistoryService) ListHistory(ctx context.Context, actingUserID, taskID int64) ([]domain.TaskHistory, error) {
	if _, _, err := s.tasks.GetVisibleTask(ctx, actingUserID, taskID); err != nil {
		return nil, err
	}

	return s.history.ListByTask(ctx, taskID)
}
