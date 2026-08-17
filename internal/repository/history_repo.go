package repository

import (
	"context"
	"fmt"

	"github.com/FL1NEE/basis_test_task/internal/domain"
)

type HistoryRepo struct {
	db DBTX
}

func NewHistoryRepo(db DBTX) *HistoryRepo {
	return &HistoryRepo{db: db}
}

func (r *HistoryRepo) Create(ctx context.Context, taskID, changedBy int64, changesJSON string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO task_history (task_id, changed_by, changes) VALUES (?, ?, ?)`,
		taskID, changedBy, changesJSON,
	)
	if err != nil {
		return fmt.Errorf("insert task history: %w", err)
	}
	return nil
}

func (r *HistoryRepo) ListByTask(ctx context.Context, taskID int64) ([]domain.TaskHistory, error) {
	var history []domain.TaskHistory
	err := r.db.SelectContext(ctx, &history,
		`SELECT id, task_id, changed_by, changes, created_at
		 FROM task_history WHERE task_id = ? ORDER BY created_at DESC`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list task history: %w", err)
	}
	return history, nil
}
