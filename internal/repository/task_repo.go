package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/FL1NEE/basis_test_task/internal/domain"
)

type TaskRepo struct {
	db DBTX
}

func NewTaskRepo(db DBTX) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(ctx context.Context, t *domain.Task) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		t.TeamID, t.Title, t.Description, t.Status, t.CreatedBy, t.AssigneeID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read inserted task id: %w", err)
	}
	return id, nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	var t domain.Task
	err := r.db.GetContext(ctx, &t,
		`SELECT id, team_id, title, description, status, created_by, assignee_id,
		        created_at, updated_at, closed_at, version
		 FROM tasks WHERE id = ?`,
		id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return &t, nil
}

// GetVisibleByID resolves a task only if userID is a member of its team,
// in one round trip. A task that exists but belongs to a team userID
// isn't on is indistinguishable from one that doesn't exist at all -
// both come back as domain.ErrNotFound - so callers can't enumerate
// task IDs belonging to other teams by diffing error responses.
func (r *TaskRepo) GetVisibleByID(ctx context.Context, taskID, userID int64) (*domain.Task, domain.Role, error) {
	var row struct {
		domain.Task
		Role domain.Role `db:"member_role"`
	}
	err := r.db.GetContext(ctx, &row,
		`SELECT t.id, t.team_id, t.title, t.description, t.status, t.created_by,
		        t.assignee_id, t.created_at, t.updated_at, t.closed_at, t.version,
		        tm.role AS member_role
		 FROM tasks t
		 JOIN team_members tm ON tm.team_id = t.team_id AND tm.user_id = ?
		 WHERE t.id = ?`,
		userID, taskID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", domain.ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("get visible task: %w", err)
	}
	return &row.Task, row.Role, nil
}

type TaskFilter struct {
	TeamID     int64
	Status     *domain.TaskStatus
	AssigneeID *int64
	Limit      int
	Offset     int
}

func (r *TaskRepo) List(ctx context.Context, f TaskFilter) ([]domain.Task, error) {
	query := strings.Builder{}
	query.WriteString(`SELECT id, team_id, title, description, status, created_by, assignee_id,
	                           created_at, updated_at, closed_at, version
	                    FROM tasks WHERE team_id = ?`)
	args := []any{f.TeamID}

	if f.Status != nil {
		query.WriteString(` AND status = ?`)
		args = append(args, *f.Status)
	}
	if f.AssigneeID != nil {
		query.WriteString(` AND assignee_id = ?`)
		args = append(args, *f.AssigneeID)
	}
	query.WriteString(` ORDER BY created_at DESC LIMIT ? OFFSET ?`)
	args = append(args, f.Limit, f.Offset)

	var tasks []domain.Task
	if err := r.db.SelectContext(ctx, &tasks, query.String(), args...); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

// UpdateWithVersion is a CAS: it only applies if the row is still at
// expectedVersion, otherwise domain.ErrVersionMismatch.
func (r *TaskRepo) UpdateWithVersion(ctx context.Context, id int64, expectedVersion int, t *domain.Task) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tasks
		 SET title = ?, description = ?, status = ?, assignee_id = ?, closed_at = ?, version = version + 1
		 WHERE id = ? AND version = ?`,
		t.Title, t.Description, t.Status, t.AssigneeID, t.ClosedAt, id, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read update result: %w", err)
	}
	if affected == 0 {
		return domain.ErrVersionMismatch
	}
	return nil
}
