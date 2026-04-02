package service

import (
	"context"
	"errors"
	"html"
	"strings"
	"time"

	"tip2_pr7/services/tasks/internal/repository"
)

var ErrNotFound = repository.ErrNotFound

type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	Done        bool   `json:"done"`
	CreatedAt   string `json:"created_at"`
}

type CreateTaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"due_date"`
}

type UpdateTaskInput struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	Done        *bool   `json:"done,omitempty"`
}

type TaskService struct {
	repo repository.TaskRepository
}

func New(repo repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) Create(ctx context.Context, input CreateTaskInput) (Task, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Task{}, errors.New("title is required")
	}

	dueDate, err := parseOptionalDate(input.DueDate)
	if err != nil {
		return Task{}, err
	}

	task, err := s.repo.Create(ctx, repository.CreateTaskParams{
		Title:       title,
		Description: sanitizePlainText(input.Description),
		DueDate:     dueDate,
	})
	if err != nil {
		return Task{}, err
	}

	return toTaskDTO(task), nil
}

func (s *TaskService) List(ctx context.Context) ([]Task, error) {
	tasks, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	return toTaskDTOList(tasks), nil
}

func (s *TaskService) SearchByTitle(ctx context.Context, title string) ([]Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("title query parameter is required")
	}

	tasks, err := s.repo.SearchByTitle(ctx, title)
	if err != nil {
		return nil, err
	}

	return toTaskDTOList(tasks), nil
}

func (s *TaskService) Get(ctx context.Context, id string) (Task, error) {
	task, err := s.repo.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}

	return toTaskDTO(task), nil
}

func (s *TaskService) Update(ctx context.Context, id string, input UpdateTaskInput) (Task, error) {
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}

	if input.Title != nil {
		current.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		current.Description = sanitizePlainText(*input.Description)
	}
	if input.DueDate != nil {
		dueDate, err := parseOptionalDate(*input.DueDate)
		if err != nil {
			return Task{}, err
		}
		current.DueDate = dueDate
	}
	if input.Done != nil {
		current.Done = *input.Done
	}

	if strings.TrimSpace(current.Title) == "" {
		return Task{}, errors.New("title is required")
	}

	updated, err := s.repo.Update(ctx, id, repository.UpdateTaskParams{
		Title:       current.Title,
		Description: current.Description,
		DueDate:     current.DueDate,
		Done:        current.Done,
	})
	if err != nil {
		return Task{}, err
	}

	return toTaskDTO(updated), nil
}

func (s *TaskService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func parseOptionalDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, errors.New("due_date must be in YYYY-MM-DD format")
	}

	return &parsed, nil
}

func sanitizePlainText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return html.EscapeString(value)
}

func toTaskDTO(task repository.Task) Task {
	result := Task{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Done:        task.Done,
		CreatedAt:   task.CreatedAt.UTC().Format(time.RFC3339),
	}

	if task.DueDate != nil {
		result.DueDate = task.DueDate.UTC().Format("2006-01-02")
	}

	return result
}

func toTaskDTOList(tasks []repository.Task) []Task {
	result := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, toTaskDTO(task))
	}
	return result
}
