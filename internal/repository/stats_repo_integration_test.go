//go:build integration

// This is the one integration test required by the spec: it exercises
// the SQL report end to end against a real MySQL instance (via
// testcontainers-go), not a mock. Run it explicitly with:
//
//	go test -tags=integration ./internal/repository/... -run Stats -v
//
// It needs a working Docker daemon and is excluded from the default
// `go test ./...` run so the rest of the suite stays fast and
// Docker-independent.
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/FL1NEE/basis_test_task/internal/repository"
	"github.com/jmoiron/sqlx"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
)

func TestStatsRepo_GetTeamStats(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container, err := tcmysql.Run(ctx, "mysql:8.0",
		tcmysql.WithDatabase("bazis_test"),
		tcmysql.WithUsername("app"),
		tcmysql.WithPassword("app"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate mysql container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "parseTime=true&multiStatements=true")
	if err != nil {
		t.Fatalf("build dsn: %v", err)
	}

	db, err := repository.Connect(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := repository.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	teamID := seedStatsFixture(t, db)

	statsRepo := repository.NewStatsRepo(db)
	stats, err := statsRepo.GetTeamStats(ctx, teamID)
	if err != nil {
		t.Fatalf("GetTeamStats: %v", err)
	}

	if got, want := stats.TasksByStatus["done"], 2; got != want {
		t.Errorf("done count = %d, want %d", got, want)
	}
	if got, want := stats.TasksByStatus["todo"], 1; got != want {
		t.Errorf("todo count = %d, want %d", got, want)
	}
	if got, want := stats.TasksByStatus["in_progress"], 1; got != want {
		t.Errorf("in_progress count = %d, want %d", got, want)
	}
	if got, want := stats.TotalComments, 3; got != want {
		t.Errorf("total comments = %d, want %d", got, want)
	}
	if len(stats.TopAssignees) != 1 {
		t.Fatalf("expected exactly 1 assignee with closed tasks in the last 30 days, got %d", len(stats.TopAssignees))
	}
	if got, want := stats.TopAssignees[0].ClosedCount, 2; got != want {
		t.Errorf("top assignee closed_count = %d, want %d", got, want)
	}
	if stats.AvgCloseHours == nil {
		t.Fatal("expected avg_close_hours to be populated")
	}
	if *stats.AvgCloseHours <= 0 {
		t.Errorf("avg_close_hours = %f, want > 0", *stats.AvgCloseHours)
	}
}

// seedStatsFixture creates one team with two users: a creator/owner and
// an assignee. It inserts 4 tasks (2 done within the last 30 days, 1
// todo, 1 in_progress), 3 comments, and one closed task from more than
// 30 days ago that must NOT count toward the "last 30 days" metric -
// this is the case that would silently pass with a naive query missing
// the date filter.
func seedStatsFixture(t *testing.T, db *sqlx.DB) int64 {
	t.Helper()

	ownerID := mustExecLastID(t, db, `INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`,
		"owner@example.com", "hash", "Owner")
	assigneeID := mustExecLastID(t, db, `INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`,
		"assignee@example.com", "hash", "Assignee")

	teamID := mustExecLastID(t, db, `INSERT INTO teams (name, created_by) VALUES (?, ?)`,
		"Stats Team", ownerID)
	mustExec(t, db, `INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, 'owner')`, teamID, ownerID)
	mustExec(t, db, `INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, 'member')`, teamID, assigneeID)

	// Two tasks closed within the last 30 days by the assignee.
	for i := 0; i < 2; i++ {
		taskID := mustExecLastID(t, db,
			`INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, closed_at)
			 VALUES (?, ?, 'done', ?, ?, NOW() - INTERVAL 1 DAY, NOW())`,
			teamID, "done task", ownerID, assigneeID)
		mustExec(t, db, `INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`,
			taskID, assigneeID, "comment")
	}

	// One task closed 60 days ago: must be excluded from the 30-day window.
	mustExec(t, db,
		`INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, closed_at)
		 VALUES (?, ?, 'done', ?, ?, NOW() - INTERVAL 61 DAY, NOW() - INTERVAL 60 DAY)`,
		teamID, "old done task", ownerID, assigneeID)

	todoID := mustExecLastID(t, db,
		`INSERT INTO tasks (team_id, title, status, created_by) VALUES (?, ?, 'todo', ?)`,
		teamID, "todo task", ownerID)
	mustExec(t, db, `INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`,
		todoID, ownerID, "comment")

	inProgressID := mustExecLastID(t, db,
		`INSERT INTO tasks (team_id, title, status, created_by) VALUES (?, ?, 'in_progress', ?)`,
		teamID, "in progress task", ownerID)
	mustExec(t, db, `INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`,
		inProgressID, ownerID, "comment")

	return teamID
}

func mustExec(t *testing.T, db *sqlx.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func mustExecLastID(t *testing.T, db *sqlx.DB, query string, args ...any) int64 {
	t.Helper()
	res, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}
