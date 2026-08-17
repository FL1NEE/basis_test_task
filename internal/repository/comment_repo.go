package repository

import (
	"context"
	"fmt"

	"github.com/FL1NEE/basis_test_task/internal/domain"
)

type CommentRepo struct {
	db DBTX
}

func NewCommentRepo(db DBTX) *CommentRepo {
	return &CommentRepo{db: db}
}

func (r *CommentRepo) Create(ctx context.Context, taskID, userID int64, content string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`,
		taskID, userID, content,
	)
	if err != nil {
		return 0, fmt.Errorf("insert comment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read inserted comment id: %w", err)
	}
	return id, nil
}

func (r *CommentRepo) ListByTask(ctx context.Context, taskID int64) ([]domain.TaskComment, error) {
	var comments []domain.TaskComment
	err := r.db.SelectContext(ctx, &comments,
		`SELECT id, task_id, user_id, content, created_at
		 FROM task_comments WHERE task_id = ? ORDER BY created_at ASC`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	return comments, nil
}
