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

// UpdateWithVersion applies the given field values only if the row is
// still at expectedVersion, bumping the version by one. It reports
// domain.ErrVersionMismatch if another update won the race in between the
// caller reading the task and calling this method - the standard
// optimistic-concurrency pattern, so a losing writer never silently
// clobbers a winning one.
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
