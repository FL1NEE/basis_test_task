package service

import (
	"context"

	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/FL1NEE/basis_test_task/internal/repository"
)

type StatsService struct {
	stats   *repository.StatsRepo
	teamSvc *TeamService
}

func NewStatsService(stats *repository.StatsRepo, teamSvc *TeamService) *StatsService {
	return &StatsService{stats: stats, teamSvc: teamSvc}
}

func (s *StatsService) GetTeamStats(ctx context.Context, actingUserID, teamID int64) (*domain.TeamStats, error) {
	role, err := s.teamSvc.RequireMembership(ctx, teamID, actingUserID)
	if err != nil {
		return nil, err
	}
	if role != domain.RoleOwner && role != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}

	return s.stats.GetTeamStats(ctx, teamID)
}
