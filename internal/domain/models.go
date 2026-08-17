package domain

import "time"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone:
		return true
	default:
		return false
	}
}

type User struct {
	ID           int64     `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Name         string    `json:"name" db:"name"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Team struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedBy int64     `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	// Role is the current user's role in this team. Populated by queries
	// that list "my teams"; not a column on the teams table itself.
	Role Role `json:"role,omitempty" db:"role"`
}

type TeamMember struct {
	TeamID    int64     `json:"team_id" db:"team_id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Role      Role      `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Task struct {
	ID          int64      `json:"id" db:"id"`
	TeamID      int64      `json:"team_id" db:"team_id"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description" db:"description"`
	Status      TaskStatus `json:"status" db:"status"`
	CreatedBy   int64      `json:"created_by" db:"created_by"`
	AssigneeID  *int64     `json:"assignee_id" db:"assignee_id"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at" db:"closed_at"`
	Version     int        `json:"version" db:"version"`
}

type TaskHistory struct {
	ID        int64     `json:"id" db:"id"`
	TaskID    int64     `json:"task_id" db:"task_id"`
	ChangedBy int64     `json:"changed_by" db:"changed_by"`
	Changes   string    `json:"changes" db:"changes"` // raw JSON, see FieldChange
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// FieldChange is the shape stored in task_history.changes for every field
// that was actually modified by an update.
type FieldChange struct {
	Old any `json:"old"`
	New any `json:"new"`
}

type TaskComment struct {
	ID        int64     `json:"id" db:"id"`
	TaskID    int64     `json:"task_id" db:"task_id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type TeamStats struct {
	TeamID        int64              `json:"team_id"`
	TasksByStatus map[TaskStatus]int `json:"tasks_by_status"`
	TopAssignees  []AssigneeStat     `json:"top_assignees"`
	AvgCloseHours *float64           `json:"avg_close_hours"`
	TotalComments int                `json:"total_comments"`
}

type AssigneeStat struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	ClosedCount int    `json:"closed_count"`
}
