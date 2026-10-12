package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"task-tracker/internal/models"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM projects
		ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]models.Project, 0)
	for rows.Next() {
		var project models.Project
		if err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&project.CreatedAt,
			&project.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, nil
}

func (r *Repository) GetProject(ctx context.Context, id models.ProjectID) (models.Project, error) {
	var project models.Project
	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM projects
		WHERE id = $1`, int64(id)).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Project{}, ErrNotFound
	}
	if err != nil {
		return models.Project{}, fmt.Errorf("get project: %w", err)
	}
	return project, nil
}

func (r *Repository) CreateProject(ctx context.Context, input models.ProjectInput) (models.Project, error) {
	var project models.Project
	err := r.db.QueryRow(ctx, `
		INSERT INTO projects (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, created_at, updated_at`,
		input.Name, input.Description,
	).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		return models.Project{}, fmt.Errorf("create project: %w", err)
	}
	return project, nil
}

func (r *Repository) UpdateProject(ctx context.Context, id models.ProjectID, input models.ProjectInput) (models.Project, error) {
	var project models.Project
	err := r.db.QueryRow(ctx, `
		UPDATE projects
		SET name = $2, description = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, name, description, created_at, updated_at`,
		int64(id), input.Name, input.Description,
	).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Project{}, ErrNotFound
	}
	if err != nil {
		return models.Project{}, fmt.Errorf("update project: %w", err)
	}
	return project, nil
}

func (r *Repository) DeleteProject(ctx context.Context, id models.ProjectID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM projects WHERE id = $1`, int64(id))
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListTasks(ctx context.Context, projectID *models.ProjectID, status models.TaskStatus) ([]models.Task, error) {
	query := `
		SELECT id, project_id, title, description, status, created_at, updated_at
		FROM tasks
		WHERE ($1::bigint IS NULL OR project_id = $1)
		  AND ($2::text = '' OR status = $2)
		ORDER BY created_at DESC, id DESC`

	var projectIDParam any
	if projectID != nil {
		projectIDParam = int64(*projectID)
	}

	rows, err := r.db.Query(ctx, query, projectIDParam, string(status))
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]models.Task, 0)
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func (r *Repository) GetTask(ctx context.Context, id models.TaskID) (models.Task, error) {
	var task models.Task
	err := r.db.QueryRow(ctx, `
		SELECT id, project_id, title, description, status, created_at, updated_at
		FROM tasks
		WHERE id = $1`, int64(id)).Scan(
		&task.ID,
		&task.ProjectID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Task{}, ErrNotFound
	}
	if err != nil {
		return models.Task{}, fmt.Errorf("get task: %w", err)
	}
	return task, nil
}

func (r *Repository) CreateTask(ctx context.Context, input models.TaskInput) (models.Task, error) {
	var task models.Task
	err := r.db.QueryRow(ctx, `
		INSERT INTO tasks (project_id, title, description, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, project_id, title, description, status, created_at, updated_at`,
		int64(input.ProjectID), input.Title, input.Description, string(input.Status),
	).Scan(
		&task.ID,
		&task.ProjectID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		return models.Task{}, fmt.Errorf("create task: %w", err)
	}
	return task, nil
}

func (r *Repository) UpdateTask(ctx context.Context, id models.TaskID, input models.TaskInput) (models.Task, error) {
	var task models.Task
	err := r.db.QueryRow(ctx, `
		UPDATE tasks
		SET project_id = $2, title = $3, description = $4, status = $5, updated_at = now()
		WHERE id = $1
		RETURNING id, project_id, title, description, status, created_at, updated_at`,
		int64(id), int64(input.ProjectID), input.Title, input.Description, string(input.Status),
	).Scan(
		&task.ID,
		&task.ProjectID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Task{}, ErrNotFound
	}
	if err != nil {
		return models.Task{}, fmt.Errorf("update task: %w", err)
	}
	return task, nil
}

func (r *Repository) DeleteTask(ctx context.Context, id models.TaskID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, int64(id))
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
