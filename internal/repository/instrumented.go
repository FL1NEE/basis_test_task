package repository

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"time"

	"github.com/FL1NEE/basis_test_task/internal/metrics"
)

// Instrument wraps a DBTX so every query it runs is recorded via
// internal/metrics, regardless of whether it's the plain pool or a
// transaction. Centralizing this at the DBTX boundary means individual
// repository methods don't need to know about metrics.
func Instrument(db DBTX) DBTX {
	return &instrumentedDB{inner: db}
}

type instrumentedDB struct {
	inner DBTX
}

func (i *instrumentedDB) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	start := time.Now()
	err := i.inner.GetContext(ctx, dest, query, args...)
	metrics.RecordDBQuery(queryOperation(query), time.Since(start))
	return err
}

func (i *instrumentedDB) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	start := time.Now()
	err := i.inner.SelectContext(ctx, dest, query, args...)
	metrics.RecordDBQuery(queryOperation(query), time.Since(start))
	return err
}

func (i *instrumentedDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	res, err := i.inner.ExecContext(ctx, query, args...)
	metrics.RecordDBQuery(queryOperation(query), time.Since(start))
	return res, err
}

var (
	verbRe  = regexp.MustCompile(`(?i)^\s*(SELECT|INSERT|UPDATE|DELETE|WITH)`)
	tableRe = regexp.MustCompile("(?is)\\b(?:FROM|INTO|UPDATE|JOIN)\\s+`?(\\w+)")
)

// queryOperation derives a low-cardinality metric label like
// "select_tasks" from raw SQL, so dashboards can group latency/count by
// verb+table without every call site having to pass one explicitly.
func queryOperation(query string) string {
	verb := "query"
	if m := verbRe.FindStringSubmatch(query); m != nil {
		verb = strings.ToLower(m[1])
		if verb == "with" {
			verb = "select" // CTEs are reporting selects in this codebase
		}
	}

	table := "unknown"
	if m := tableRe.FindStringSubmatch(query); m != nil {
		table = strings.ToLower(m[1])
	}

	return verb + "_" + table
}
