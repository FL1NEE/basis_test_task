package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/redis/go-redis/v9"
)

// TaskListCache caches the result of a filtered task listing per team.
// Invalidation uses a per-team generation counter instead of scanning and
// deleting keys by pattern: every cache key embeds the team's current
// generation, and invalidating a team is a single atomic INCR. Old
// entries are simply never looked up again and expire on their own TTL.
type TaskListCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewTaskListCache(client *redis.Client, ttl time.Duration) *TaskListCache {
	return &TaskListCache{client: client, ttl: ttl}
}

func (c *TaskListCache) generationKey(teamID int64) string {
	return fmt.Sprintf("tasks:team:%d:gen", teamID)
}

func (c *TaskListCache) currentGeneration(ctx context.Context, teamID int64) (int64, error) {
	gen, err := c.client.Get(ctx, c.generationKey(teamID)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read cache generation: %w", err)
	}
	return gen, nil
}

func (c *TaskListCache) entryKey(teamID int64, generation int64, status, assigneeID string, limit, offset int) string {
	return fmt.Sprintf("tasks:team:%d:gen:%d:status:%s:assignee:%s:limit:%d:offset:%d",
		teamID, generation, status, assigneeID, limit, offset)
}

// Get returns the cached task list for the given filter combination, or
// (nil, false, nil) on a cache miss.
func (c *TaskListCache) Get(ctx context.Context, teamID int64, status, assigneeID string, limit, offset int) ([]domain.Task, bool, error) {
	generation, err := c.currentGeneration(ctx, teamID)
	if err != nil {
		return nil, false, err
	}

	raw, err := c.client.Get(ctx, c.entryKey(teamID, generation, status, assigneeID, limit, offset)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read cached task list: %w", err)
	}

	var tasks []domain.Task
	if err := json.Unmarshal(raw, &tasks); err != nil {
		return nil, false, fmt.Errorf("decode cached task list: %w", err)
	}
	return tasks, true, nil
}

func (c *TaskListCache) Set(ctx context.Context, teamID int64, status, assigneeID string, limit, offset int, tasks []domain.Task) error {
	generation, err := c.currentGeneration(ctx, teamID)
	if err != nil {
		return err
	}

	raw, err := json.Marshal(tasks)
	if err != nil {
		return fmt.Errorf("encode task list for cache: %w", err)
	}

	key := c.entryKey(teamID, generation, status, assigneeID, limit, offset)
	if err := c.client.Set(ctx, key, raw, c.ttl).Err(); err != nil {
		return fmt.Errorf("write task list cache: %w", err)
	}
	return nil
}

// InvalidateTeam bumps the team's cache generation, instantly making
// every previously cached list for that team unreachable regardless of
// which filter combination produced it.
func (c *TaskListCache) InvalidateTeam(ctx context.Context, teamID int64) error {
	if err := c.client.Incr(ctx, c.generationKey(teamID)).Err(); err != nil {
		return fmt.Errorf("bump cache generation: %w", err)
	}
	return nil
}
