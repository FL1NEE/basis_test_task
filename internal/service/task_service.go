package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FL1NEE/basis_test_task/internal/cache"
	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/FL1NEE/basis_test_task/internal/repository"
	"github.com/jmoiron/sqlx"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

type TaskService struct {
	db      *sqlx.DB
	tasks   *repository.TaskRepo
	history *repository.HistoryRepo
	teamSvc *TeamService
	cache   *cache.CachedTaskList
}

func NewTaskService(
	db *sqlx.DB,
	tasks *repository.TaskRepo,
	history *repository.HistoryRepo,
	teamSvc *TeamService,
	taskCache *cache.CachedTaskList,
) *TaskService {
	return &TaskService{db: db, tasks: tasks, history: history, teamSvc: teamSvc, cache: taskCache}
}

func (s *TaskService) CreateTask(ctx context.Context, actingUserID, teamID int64, title string, description *string, assigneeID *int64) (*domain.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrValidation)
	}

	if _, err := s.teamSvc.RequireMembership(ctx, teamID, actingUserID); err != nil {
		return nil, err
	}

	if assigneeID != nil {
		if _, err := s.teamSvc.RequireMembership(ctx, teamID, *assigneeID); err != nil {
			return nil, fmt.Errorf("%w: assignee must be a member of the team", domain.ErrValidation)
		}
	}

	task := &domain.Task{
		TeamID:      teamID,
		Title:       title,
		Description: description,
		Status:      domain.TaskStatusTodo,
		CreatedBy:   actingUserID,
		AssigneeID:  assigneeID,
	}

	changesJSON, err := json.Marshal(creationChanges(task))
	if err != nil {
		return nil, fmt.Errorf("encode task history: %w", err)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	txDB := repository.Instrument(tx)
	taskRepo := repository.NewTaskRepo(txDB)
	historyRepo := repository.NewHistoryRepo(txDB)

	id, err := taskRepo.Create(ctx, task)
	if err != nil {
		return nil, err
	}
	if err := historyRepo.Create(ctx, id, actingUserID, string(changesJSON)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	if err := s.cache.InvalidateTeam(ctx, teamID); err != nil {
		return nil, err
	}

	return s.tasks.GetByID(ctx, id)
}

// creationChanges records the task's initial field values as a
// {old: null, new: value} history entry, so a task's history is a
// complete trail from creation onward rather than starting at its first
// edit.
func creationChanges(t *domain.Task) map[string]domain.FieldChange {
	changes := map[string]domain.FieldChange{
		"title":  {Old: nil, New: t.Title},
		"status": {Old: nil, New: t.Status},
	}
	if t.Description != nil {
		changes["description"] = domain.FieldChange{Old: nil, New: *t.Description}
	}
	if t.AssigneeID != nil {
		changes["assignee_id"] = domain.FieldChange{Old: nil, New: *t.AssigneeID}
	}
	return changes
}

type ListTasksParams struct {
	TeamID     int64
	Status     *domain.TaskStatus
	AssigneeID *int64
	Limit      int
	Offset     int
}

func (s *TaskService) ListTasks(ctx context.Context, actingUserID int64, p ListTasksParams) ([]domain.Task, error) {
	if _, err := s.teamSvc.RequireMembership(ctx, p.TeamID, actingUserID); err != nil {
		return nil, err
	}

	if p.Status != nil && !p.Status.Valid() {
		return nil, fmt.Errorf("%w: invalid status filter", domain.ErrValidation)
	}
	if p.Limit <= 0 {
		p.Limit = defaultListLimit
	}
	if p.Limit > maxListLimit {
		p.Limit = maxListLimit
	}
	if p.Offset < 0 {
		p.Offset = 0
	}

	statusKey, assigneeKey := cacheFilterKeys(p.Status, p.AssigneeID)

	if cached, hit, err := s.cache.Get(ctx, p.TeamID, statusKey, assigneeKey, p.Limit, p.Offset); err != nil {
		return nil, err
	} else if hit {
		return cached, nil
	}

	tasks, err := s.tasks.List(ctx, repository.TaskFilter{
		TeamID:     p.TeamID,
		Status:     p.Status,
		AssigneeID: p.AssigneeID,
		Limit:      p.Limit,
		Offset:     p.Offset,
	})
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, p.TeamID, statusKey, assigneeKey, p.Limit, p.Offset, tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func cacheFilterKeys(status *domain.TaskStatus, assigneeID *int64) (string, string) {
	statusKey := "-"
	if status != nil {
		statusKey = string(*status)
	}
	assigneeKey := "-"
	if assigneeID != nil {
		assigneeKey = strconv.FormatInt(*assigneeID, 10)
	}
	return statusKey, assigneeKey
}

// TaskPatch: nil field means "leave as-is".
type TaskPatch struct {
	Title       *string
	Description *string
	Status      *domain.TaskStatus
	AssigneeID  *int64
}

func (p TaskPatch) empty() bool {
	return p.Title == nil && p.Description == nil && p.Status == nil && p.AssigneeID == nil
}

func (s *TaskService) UpdateTask(ctx context.Context, actingUserID, taskID int64, expectedVersion int, patch TaskPatch) (*domain.Task, error) {
	if patch.empty() {
		return nil, fmt.Errorf("%w: no fields to update", domain.ErrValidation)
	}

	current, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	role, err := requireTaskTeamMembership(s.teamSvc, ctx, current.TeamID, actingUserID)
	if err != nil {
		return nil, err
	}

	switch taskEditAccess(role, current, actingUserID) {
	case taskAccessFull:
		if patch.Status != nil && !patch.Status.Valid() {
			return nil, fmt.Errorf("%w: invalid status", domain.ErrValidation)
		}
		if patch.AssigneeID != nil {
			if _, err := s.teamSvc.RequireMembership(ctx, current.TeamID, *patch.AssigneeID); err != nil {
				return nil, fmt.Errorf("%w: assignee must be a member of the team", domain.ErrValidation)
			}
		}
	case taskAccessStatusOnly:
		if patch.Title != nil || patch.Description != nil || patch.AssigneeID != nil {
			return nil, domain.ErrForbidden
		}
		if patch.Status == nil || !patch.Status.Valid() {
			return nil, fmt.Errorf("%w: invalid status", domain.ErrValidation)
		}
	default:
		return nil, domain.ErrForbidden
	}

	updated := *current
	if patch.Title != nil {
		updated.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Description != nil {
		updated.Description = patch.Description
	}
	if patch.AssigneeID != nil {
		updated.AssigneeID = patch.AssigneeID
	}
	if patch.Status != nil {
		updated.Status = *patch.Status
		if updated.Status == domain.TaskStatusDone && current.Status != domain.TaskStatusDone {
			now := time.Now().UTC()
			updated.ClosedAt = &now
		} else if updated.Status != domain.TaskStatusDone && current.Status == domain.TaskStatusDone {
			updated.ClosedAt = nil
		}
	}

	changes := diffTask(current, &updated)
	if len(changes) == 0 {
		return current, nil
	}
	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return nil, fmt.Errorf("encode task history: %w", err)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	txDB := repository.Instrument(tx)
	taskRepo := repository.NewTaskRepo(txDB)
	historyRepo := repository.NewHistoryRepo(txDB)

	if err := taskRepo.UpdateWithVersion(ctx, taskID, expectedVersion, &updated); err != nil {
		return nil, err
	}
	if err := historyRepo.Create(ctx, taskID, actingUserID, string(changesJSON)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	if err := s.cache.InvalidateTeam(ctx, current.TeamID); err != nil {
		return nil, err
	}

	return s.tasks.GetByID(ctx, taskID)
}

func diffTask(old, new_ *domain.Task) map[string]domain.FieldChange {
	changes := make(map[string]domain.FieldChange)

	if old.Title != new_.Title {
		changes["title"] = domain.FieldChange{Old: old.Title, New: new_.Title}
	}
	if !strPtrEqual(old.Description, new_.Description) {
		changes["description"] = domain.FieldChange{Old: old.Description, New: new_.Description}
	}
	if old.Status != new_.Status {
		changes["status"] = domain.FieldChange{Old: old.Status, New: new_.Status}
	}
	if !int64PtrEqual(old.AssigneeID, new_.AssigneeID) {
		changes["assignee_id"] = domain.FieldChange{Old: old.AssigneeID, New: new_.AssigneeID}
	}

	return changes
}

func strPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func int64PtrEqual(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
