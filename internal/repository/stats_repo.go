package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/FL1NEE/basis_test_task/internal/domain"
)

type StatsRepo struct {
	db DBTX
}

func NewStatsRepo(db DBTX) *StatsRepo {
	return &StatsRepo{db: db}
}

type statsRow struct {
	StatusJSON       sql.NullString  `db:"status_json"`
	TopAssigneesJSON sql.NullString  `db:"top_assignees_json"`
	AvgCloseSeconds  sql.NullFloat64 `db:"avg_close_seconds"`
	TotalComments    int             `db:"total_comments"`
}

// statsQuery computes every metric for a single team in one round trip:
// task counts per status, the top-3 assignees by tasks closed in the last
// 30 days, the average time-to-close, and the total comment count on the
// team's tasks. Each CTE is aggregated to a single row and the outer
// SELECT stitches them together with JSON_OBJECTAGG/JSON_ARRAYAGG so the
// whole report comes back as one row - no N+1, no application-side
// fan-out queries.
const statsQuery = `
WITH status_counts AS (
    SELECT status, COUNT(*) AS cnt
    FROM tasks
    WHERE team_id = ?
    GROUP BY status
),
closed_last_30d AS (
    SELECT assignee_id, COUNT(*) AS closed_count
    FROM tasks
    WHERE team_id = ?
      AND status = 'done'
      AND closed_at IS NOT NULL
      AND closed_at >= (NOW() - INTERVAL 30 DAY)
      AND assignee_id IS NOT NULL
    GROUP BY assignee_id
),
top_assignees AS (
    SELECT c.assignee_id, u.name, c.closed_count
    FROM closed_last_30d c
    JOIN users u ON u.id = c.assignee_id
    ORDER BY c.closed_count DESC
    LIMIT 3
),
avg_close AS (
    SELECT AVG(TIMESTAMPDIFF(SECOND, created_at, closed_at)) AS avg_seconds
    FROM tasks
    WHERE team_id = ? AND closed_at IS NOT NULL
),
comment_count AS (
    SELECT COUNT(*) AS cnt
    FROM task_comments tc
    JOIN tasks t ON t.id = tc.task_id
    WHERE t.team_id = ?
)
SELECT
    (SELECT JSON_OBJECTAGG(status, cnt) FROM status_counts) AS status_json,
    (SELECT JSON_ARRAYAGG(JSON_OBJECT('user_id', assignee_id, 'name', name, 'closed_count', closed_count))
       FROM top_assignees) AS top_assignees_json,
    (SELECT avg_seconds FROM avg_close) AS avg_close_seconds,
    (SELECT cnt FROM comment_count) AS total_comments
`

func (r *StatsRepo) GetTeamStats(ctx context.Context, teamID int64) (*domain.TeamStats, error) {
	var row statsRow
	err := r.db.GetContext(ctx, &row, statsQuery, teamID, teamID, teamID, teamID)
	if err != nil {
		return nil, fmt.Errorf("query team stats: %w", err)
	}

	stats := &domain.TeamStats{
		TeamID: teamID,
		TasksByStatus: map[domain.TaskStatus]int{
			domain.TaskStatusTodo:       0,
			domain.TaskStatusInProgress: 0,
			domain.TaskStatusDone:       0,
		},
		TotalComments: row.TotalComments,
	}

	if row.StatusJSON.Valid {
		var counts map[domain.TaskStatus]int
		if err := json.Unmarshal([]byte(row.StatusJSON.String), &counts); err != nil {
			return nil, fmt.Errorf("decode status counts: %w", err)
		}
		for status, count := range counts {
			stats.TasksByStatus[status] = count
		}
	}

	if row.TopAssigneesJSON.Valid {
		if err := json.Unmarshal([]byte(row.TopAssigneesJSON.String), &stats.TopAssignees); err != nil {
			return nil, fmt.Errorf("decode top assignees: %w", err)
		}
	}

	if row.AvgCloseSeconds.Valid {
		avgHours := row.AvgCloseSeconds.Float64 / 3600
		stats.AvgCloseHours = &avgHours
	}

	return stats, nil
}
