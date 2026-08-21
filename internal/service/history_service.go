package service

import (
	"context"

	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/FL1NEE/basis_test_task/internal/repository"
)

type HistoryService struct {
	history *repository.HistoryRepo
	tasks   *repository.TaskRepo
	teamSvc *TeamService
}

func NewHistoryService(history *repository.HistoryRepo, tasks *repository.TaskRepo, teamSvc *TeamService) *HistoryService {
	return &HistoryService{history: history, tasks: tasks, teamSvc: teamSvc}
}

func (s *HistoryService) ListHistory(ctx context.Context, actingUserID, taskID int64) ([]domain.TaskHistory, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := requireTaskTeamMembership(s.teamSvc, ctx, task.TeamID, actingUserID); err != nil {
		return nil, err
	}

	return s.history.ListByTask(ctx, taskID)
}
