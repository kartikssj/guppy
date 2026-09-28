package gtasks

import (
	"context"
	"fmt"
	"net/http"

	"google.golang.org/api/option"
	tasks "google.golang.org/api/tasks/v1"
)

// Service is the subset of the Google Tasks API that guppy uses.
// It is an interface so the TUI can be tested against a fake.
type Service interface {
	Lists(ctx context.Context) ([]TaskList, error)
	Tasks(ctx context.Context, listID string) ([]Task, error)
	Add(ctx context.Context, listID, title, parentID string) (Task, error)
	Rename(ctx context.Context, listID, taskID, title string) error
	SetCompleted(ctx context.Context, listID, taskID string, completed bool) error
	Delete(ctx context.Context, listID, taskID string) error
}

// New creates a Service backed by the real Google Tasks API using an
// authorized HTTP client (see internal/auth).
func New(ctx context.Context, hc *http.Client) (Service, error) {
	api, err := tasks.NewService(ctx, option.WithHTTPClient(hc))
	if err != nil {
		return nil, fmt.Errorf("creating Tasks client: %w", err)
	}
	return &googleService{api: api}, nil
}

type googleService struct {
	api *tasks.Service
}

func (g *googleService) Lists(ctx context.Context) ([]TaskList, error) {
	var out []TaskList
	err := g.api.Tasklists.List().MaxResults(100).Pages(ctx, func(page *tasks.TaskLists) error {
		for _, l := range page.Items {
			out = append(out, TaskList{ID: l.Id, Title: l.Title})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("fetching task lists: %w", err)
	}
	return out, nil
}

func (g *googleService) Tasks(ctx context.Context, listID string) ([]Task, error) {
	var out []Task
	call := g.api.Tasks.List(listID).
		ShowCompleted(true). // show crossed-out completed tasks
		ShowHidden(false).   // but not ones the user "cleared"
		ShowDeleted(false).
		MaxResults(100)
	err := call.Pages(ctx, func(page *tasks.Tasks) error {
		for _, t := range page.Items {
			out = append(out, toTask(t))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("fetching tasks: %w", err)
	}
	return out, nil
}

// Add creates a task. When parentID is non-empty the task becomes a subtask
// of that (top-level) task.
func (g *googleService) Add(ctx context.Context, listID, title, parentID string) (Task, error) {
	call := g.api.Tasks.Insert(listID, &tasks.Task{Title: title}).Context(ctx)
	if parentID != "" {
		call = call.Parent(parentID)
	}
	created, err := call.Do()
	if err != nil {
		return Task{}, fmt.Errorf("creating task: %w", err)
	}
	return toTask(created), nil
}

func (g *googleService) Rename(ctx context.Context, listID, taskID, title string) error {
	_, err := g.api.Tasks.Patch(listID, taskID, &tasks.Task{Title: title}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("renaming task: %w", err)
	}
	return nil
}

func (g *googleService) SetCompleted(ctx context.Context, listID, taskID string, completed bool) error {
	status := "needsAction"
	if completed {
		status = "completed"
	}
	_, err := g.api.Tasks.Patch(listID, taskID, &tasks.Task{Status: status}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("updating task: %w", err)
	}
	return nil
}

// Delete removes a task; Google Tasks also deletes its subtasks.
func (g *googleService) Delete(ctx context.Context, listID, taskID string) error {
	if err := g.api.Tasks.Delete(listID, taskID).Context(ctx).Do(); err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}
	return nil
}

func toTask(t *tasks.Task) Task {
	return Task{
		ID:        t.Id,
		Title:     t.Title,
		Parent:    t.Parent,
		Position:  t.Position,
		Completed: t.Status == "completed",
	}
}
