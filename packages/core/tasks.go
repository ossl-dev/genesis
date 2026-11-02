package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
)

type TaskResult struct {
	OK      bool   `json:"ok"`
	Details string `json:"details,omitempty"`
	Error   string `json:"error,omitempty"`
}
type Task struct {
	ID, Description string
	Priority        int
	DependsOn       []string
	Execute         func(context.Context) (TaskResult, error)
}
type taskState struct {
	task   Task
	status string
	result *TaskResult
}
type TaskRegistry struct {
	mu     sync.RWMutex
	runMu  sync.Mutex
	tasks  map[string]*taskState
	order  []string
	logger *log.Logger
	debug  bool
}

func NewTaskRegistry(loggers ...*log.Logger) *TaskRegistry {
	var logger *log.Logger
	if len(loggers) > 0 {
		logger = loggers[0]
	}
	return &TaskRegistry{tasks: map[string]*taskState{}, logger: logger, debug: os.Getenv("GENESIS_DEBUG") != ""}
}
func (t *TaskRegistry) logf(format string, args ...any) {
	if t.logger != nil {
		t.logger.Printf(format, args...)
	}
}
func (t *TaskRegistry) debugf(format string, args ...any) {
	if t.debug {
		t.logf(format, args...)
	}
}
func (t *TaskRegistry) Register(task Task) error {
	if task.ID == "" || task.Execute == nil {
		return fmt.Errorf("task id and executor are required")
	}
	t.mu.Lock()
	if t.tasks == nil {
		t.tasks = map[string]*taskState{}
	}
	if _, exists := t.tasks[task.ID]; exists {
		t.mu.Unlock()
		t.debugf("Task %s already registered, skipping", task.ID)
		return nil
	}
	task.DependsOn = append([]string(nil), task.DependsOn...)
	t.tasks[task.ID] = &taskState{task: task, status: "pending"}
	t.order = append(t.order, task.ID)
	t.mu.Unlock()
	t.debugf("Registering task: %s - %s", task.ID, task.Description)
	return nil
}
func (t *TaskRegistry) Has(id string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.tasks[id]
	return ok
}
func (t *TaskRegistry) Status(id string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if s := t.tasks[id]; s != nil {
		return s.status
	}
	return "not-found"
}
func (t *TaskRegistry) Result(id string) (TaskResult, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if s := t.tasks[id]; s != nil && s.result != nil {
		return *s.result, true
	}
	return TaskResult{}, false
}
func (t *TaskRegistry) IDs() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return append([]string(nil), t.order...)
}
func (t *TaskRegistry) Clear() {
	t.runMu.Lock()
	defer t.runMu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tasks = map[string]*taskState{}
	t.order = nil
}
func (t *TaskRegistry) sorted() ([]string, error) {
	order := append([]string(nil), t.order...)
	sort.SliceStable(order, func(i, j int) bool { return t.tasks[order[i]].task.Priority > t.tasks[order[j]].task.Priority })
	state := map[string]int{}
	result := []string{}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 2 {
			return nil
		}
		if state[id] == 1 {
			return fmt.Errorf("cyclic task dependency: %s", id)
		}
		s := t.tasks[id]
		if s == nil {
			return fmt.Errorf("missing task dependency: %s", id)
		}
		state[id] = 1
		for _, dep := range s.task.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		result = append(result, id)
		return nil
	}
	for _, id := range order {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return result, nil
}
func (t *TaskRegistry) ExecuteAll(ctx context.Context) (map[string]TaskResult, error) {
	t.runMu.Lock()
	defer t.runMu.Unlock()
	t.mu.Lock()
	order, err := t.sorted()
	t.mu.Unlock()
	if err != nil {
		return nil, err
	}
	results := map[string]TaskResult{}
	if err := ctx.Err(); err != nil {
		return results, err
	}
	if len(order) == 0 {
		t.debugf("No tasks to execute")
		return results, nil
	}
	t.logf("Executing %d system task(s)...", len(order))
	defer func() {
		succeeded := 0
		for _, result := range results {
			if result.OK {
				succeeded++
			}
		}
		t.logf("System tasks completed: %d succeeded, %d failed", succeeded, len(results)-succeeded)
	}()
	for _, id := range order {
		if err := ctx.Err(); err != nil {
			t.logf("System task execution interrupted: %v", err)
			return results, err
		}
		t.mu.Lock()
		s := t.tasks[id]
		if s.result != nil {
			results[id] = *s.result
			t.mu.Unlock()
			t.debugf("Task %s already finished, reusing result", id)
			continue
		}
		failed := false
		for _, dep := range s.task.DependsOn {
			if !results[dep].OK {
				failed = true
			}
		}
		s.status = "running"
		t.mu.Unlock()
		result := TaskResult{}
		if failed {
			result.Error = "Dependencies failed"
			t.logf("Skipping task %s due to failed dependencies", id)
		} else {
			t.debugf("Executing task: %s - %s", id, s.task.Description)
			result, err = executeTask(ctx, s.task)
			if err != nil {
				result = TaskResult{Error: err.Error()}
			}
		}
		if err := ctx.Err(); err != nil {
			result = TaskResult{Error: err.Error()}
		}
		t.mu.Lock()
		s.result = &result
		if result.OK {
			s.status = "completed"
		} else {
			s.status = "failed"
		}
		t.mu.Unlock()
		results[id] = result
		if result.OK {
			t.debugf("Task %s completed successfully: %s", id, result.Details)
		} else if !failed {
			reason := result.Error
			if reason == "" {
				reason = result.Details
			}
			if reason == "" {
				reason = "failed"
			}
			t.logf("Task %s failed: %s", id, reason)
		}
		if err := ctx.Err(); err != nil {
			return results, err
		}
	}
	return results, nil
}
func executeTask(ctx context.Context, task Task) (result TaskResult, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("task %s panicked: %v", task.ID, p)
		}
	}()
	return task.Execute(ctx)
}
