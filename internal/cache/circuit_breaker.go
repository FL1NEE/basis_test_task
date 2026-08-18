package cache

import (
	"context"
	"sync"
	"time"

	"github.com/FL1NEE/basis_test_task/internal/domain"
)

// CachedTaskList wraps TaskListCache with a circuit breaker: once Redis
// fails repeatedly, Get/Set/InvalidateTeam become no-ops (reporting a
// cache miss rather than an error) so callers fall back to the database
// instead of failing the request.
type CachedTaskList struct {
	cache   *TaskListCache
	breaker *CircuitBreaker
}

func NewCachedTaskList(cache *TaskListCache) *CachedTaskList {
	return &CachedTaskList{
		cache:   cache,
		breaker: NewCircuitBreaker(5, 30*time.Second),
	}
}

func (c *CachedTaskList) Get(ctx context.Context, teamID int64, status, assigneeID string, limit, offset int) ([]domain.Task, bool, error) {
	if c.breaker.IsOpen() {
		return nil, false, nil
	}

	tasks, hit, err := c.cache.Get(ctx, teamID, status, assigneeID, limit, offset)
	if err != nil {
		c.breaker.RecordFailure()
		return nil, false, nil
	}
	c.breaker.RecordSuccess()
	return tasks, hit, nil
}

func (c *CachedTaskList) Set(ctx context.Context, teamID int64, status, assigneeID string, limit, offset int, tasks []domain.Task) error {
	if c.breaker.IsOpen() {
		return nil
	}

	err := c.cache.Set(ctx, teamID, status, assigneeID, limit, offset, tasks)
	if err != nil {
		c.breaker.RecordFailure()
		return nil
	}
	c.breaker.RecordSuccess()
	return nil
}

func (c *CachedTaskList) InvalidateTeam(ctx context.Context, teamID int64) error {
	if c.breaker.IsOpen() {
		return nil
	}

	err := c.cache.InvalidateTeam(ctx, teamID)
	if err != nil {
		c.breaker.RecordFailure()
		return nil
	}
	c.breaker.RecordSuccess()
	return nil
}

type CircuitBreaker struct {
	mu          sync.RWMutex
	failures    int
	threshold   int
	timeout     time.Duration
	lastFailure time.Time
	state       string
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		timeout:   timeout,
		state:     "closed",
	}
}

func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == "open" && time.Since(cb.lastFailure) > cb.timeout {
		return false
	}
	return cb.state == "open"
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = "closed"
}
