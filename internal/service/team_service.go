package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/FL1NEE/basis_test_task/internal/repository"
	"github.com/jmoiron/sqlx"
)

type TeamService struct {
	db    *sqlx.DB
	teams *repository.TeamRepo
	users *repository.UserRepo
}

func NewTeamService(db *sqlx.DB, teams *repository.TeamRepo, users *repository.UserRepo) *TeamService {
	return &TeamService{db: db, teams: teams, users: users}
}

func (s *TeamService) CreateTeam(ctx context.Context, userID int64, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("%w: team name is required", domain.ErrValidation)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	teamRepo := repository.NewTeamRepo(repository.Instrument(tx))

	teamID, err := teamRepo.Create(ctx, name, userID)
	if err != nil {
		return 0, err
	}
	if err := teamRepo.AddMember(ctx, teamID, userID, domain.RoleOwner); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return teamID, nil
}

func (s *TeamService) ListMyTeams(ctx context.Context, userID int64) ([]domain.Team, error) {
	return s.teams.ListForUser(ctx, userID)
}

// InviteMember never grants or touches the owner role - ownership
// transfer is out of scope.
func (s *TeamService) InviteMember(ctx context.Context, actingUserID, teamID int64, targetEmail string, role domain.Role) error {
	if !role.Valid() || role == domain.RoleOwner {
		return fmt.Errorf("%w: invalid role for invite", domain.ErrValidation)
	}

	actingRole, err := s.teams.GetMemberRole(ctx, teamID, actingUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
		return err
	}
	if actingRole != domain.RoleOwner && actingRole != domain.RoleAdmin {
		return domain.ErrForbidden
	}

	target, err := s.users.GetByEmail(ctx, targetEmail)
	if err != nil {
		return err
	}

	existingRole, err := s.teams.GetMemberRole(ctx, teamID, target.ID)
	if err == nil {
		if existingRole == domain.RoleOwner {
			return domain.ErrForbidden
		}
		return s.teams.UpdateMemberRole(ctx, teamID, target.ID, role)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return err
	}

	return s.teams.AddMember(ctx, teamID, target.ID, role)
}

func (s *TeamService) RequireMembership(ctx context.Context, teamID, userID int64) (domain.Role, error) {
	role, err := s.teams.GetMemberRole(ctx, teamID, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return "", domain.ErrForbidden
	}
	return role, err
}
