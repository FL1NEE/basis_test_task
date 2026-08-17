package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/FL1NEE/basis_test_task/internal/domain"
)

type TeamRepo struct {
	db DBTX
}

func NewTeamRepo(db DBTX) *TeamRepo {
	return &TeamRepo{db: db}
}

func (r *TeamRepo) Create(ctx context.Context, name string, createdBy int64) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO teams (name, created_by) VALUES (?, ?)`,
		name, createdBy,
	)
	if err != nil {
		return 0, fmt.Errorf("insert team: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read inserted team id: %w", err)
	}
	return id, nil
}

func (r *TeamRepo) GetByID(ctx context.Context, id int64) (*domain.Team, error) {
	var t domain.Team
	err := r.db.GetContext(ctx, &t,
		`SELECT id, name, created_by, created_at FROM teams WHERE id = ?`,
		id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get team by id: %w", err)
	}
	return &t, nil
}

// ListForUser returns every team the given user belongs to, including
// their role in each, ordered by most recently created first.
func (r *TeamRepo) ListForUser(ctx context.Context, userID int64) ([]domain.Team, error) {
	var teams []domain.Team
	err := r.db.SelectContext(ctx, &teams,
		`SELECT t.id, t.name, t.created_by, t.created_at, tm.role
		 FROM teams t
		 JOIN team_members tm ON tm.team_id = t.id
		 WHERE tm.user_id = ?
		 ORDER BY t.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list teams for user: %w", err)
	}
	return teams, nil
}

func (r *TeamRepo) AddMember(ctx context.Context, teamID, userID int64, role domain.Role) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		teamID, userID, role,
	)
	if err != nil {
		return fmt.Errorf("add team member: %w", err)
	}
	return nil
}

// GetMemberRole returns the role of a user within a team, or
// domain.ErrNotFound if the user is not a member.
func (r *TeamRepo) GetMemberRole(ctx context.Context, teamID, userID int64) (domain.Role, error) {
	var role domain.Role
	err := r.db.GetContext(ctx, &role,
		`SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID, userID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get member role: %w", err)
	}
	return role, nil
}

func (r *TeamRepo) UpdateMemberRole(ctx context.Context, teamID, userID int64, role domain.Role) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE team_members SET role = ? WHERE team_id = ? AND user_id = ?`,
		role, teamID, userID,
	)
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	return nil
}
