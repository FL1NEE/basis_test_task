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

// GetTeamStats is restricted to the team's owner or admin, and always
// scoped to a single team_id - RequireMembership already guarantees the
// caller belongs to exactly this team, so there is no way to pull another
// team's numbers by guessing an id.
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
