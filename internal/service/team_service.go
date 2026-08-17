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

// CreateTeam creates the team and adds the creator as its owner in a
// single transaction: a team without its owner membership row (or vice
// versa) would be a broken state no caller could recover from.
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

	teamRepo := repository.NewTeamRepo(tx)

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

// InviteMember adds userEmail to teamID with the given role. Only an
// owner or admin may invite. The owner role can never be granted through
// this path, and neither an owner nor an admin can be re-assigned away
// from owner here - team ownership transfer is out of scope for this
// service.
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
		// Already a member: treat this as a role change, but never let
		// anyone touch the owner's membership through the invite path.
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

// RequireMembership checks that userID belongs to teamID and returns
// their role, or domain.ErrForbidden if they don't. Used by other
// services (tasks, comments, stats) so callers can branch on the role
// (e.g. creator/assignee vs. plain member permissions).
func (s *TeamService) RequireMembership(ctx context.Context, teamID, userID int64) (domain.Role, error) {
	role, err := s.teams.GetMemberRole(ctx, teamID, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return "", domain.ErrForbidden
	}
	return role, err
}
