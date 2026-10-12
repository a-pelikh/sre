package models

import "time"

type ProjectID int64

type TaskID int64

type Project struct {
	ID          ProjectID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProjectInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
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

type Task struct {
	ID          TaskID     `json:"id"`
	ProjectID   ProjectID  `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TaskInput struct {
	ProjectID   ProjectID  `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
}
