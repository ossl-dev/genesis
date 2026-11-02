package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestTasksOrderDedupAndCachedResults(t *testing.T) {
	registry := NewTaskRegistry()
	order := []string{}
	for _, task := range []Task{{ID: "low", Priority: 0}, {ID: "high", Priority: 100, DependsOn: []string{"dep"}}, {ID: "dep", Priority: -100}} {
		id := task.ID
		task.Execute = func(context.Context) (TaskResult, error) { order = append(order, id); return TaskResult{OK: true}, nil }
		if err := registry.Register(task); err != nil {
			t.Fatal(err)
		}
	}
	if err := registry.Register(Task{ID: "dep", Execute: func(context.Context) (TaskResult, error) { t.Fatal("duplicate executed"); return TaskResult{}, nil }}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		results, err := registry.ExecuteAll(context.Background())
		if err != nil || len(results) != 3 {
			t.Fatal(results, err)
		}
	}
	if !reflect.DeepEqual(order, []string{"dep", "high", "low"}) {
		t.Fatal(order)
	}
	if registry.Status("dep") != "completed" || registry.Status("missing") != "not-found" || !registry.Has("dep") {
		t.Fatal("status mismatch")
	}
	registry.Clear()
	if len(registry.IDs()) != 0 {
		t.Fatal("clear failed")
	}
}
func TestTaskGraphErrorsBeforeExecution(t *testing.T) {
	for _, cycle := range []bool{true, false} {
		registry := NewTaskRegistry()
		dep := "missing"
		if cycle {
			dep = "one"
		}
		_ = registry.Register(Task{ID: "one", DependsOn: []string{dep}, Execute: func(context.Context) (TaskResult, error) { t.Fatal("invalid graph executed"); return TaskResult{}, nil }})
		if _, err := registry.ExecuteAll(context.Background()); err == nil {
			t.Fatal("invalid graph accepted")
		}
	}
}
func TestTaskFailurePropagation(t *testing.T) {
	for _, kind := range []string{"result", "error", "panic"} {
		t.Run(kind, func(t *testing.T) {
			registry := NewTaskRegistry()
			_ = registry.Register(Task{ID: "bad", Execute: func(context.Context) (TaskResult, error) {
				switch kind {
				case "error":
					return TaskResult{}, errors.New("failed")
				case "panic":
					panic("failed")
				}
				return TaskResult{Error: "failed"}, nil
			}})
			_ = registry.Register(Task{ID: "dependent", DependsOn: []string{"bad"}, Execute: func(context.Context) (TaskResult, error) { t.Fatal("dependent executed"); return TaskResult{}, nil }})
			results, err := registry.ExecuteAll(context.Background())
			if err != nil || results["bad"].OK || results["dependent"].Error != "Dependencies failed" {
				t.Fatal(results, err)
			}
		})
	}
}
func TestTaskCancellationAndValidation(t *testing.T) {
	registry := NewTaskRegistry()
	if registry.Register(Task{}) == nil {
		t.Fatal("invalid task accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = registry.Register(Task{ID: "one", Execute: func(context.Context) (TaskResult, error) { t.Fatal("canceled task ran"); return TaskResult{}, nil }})
	if _, err := registry.ExecuteAll(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestTaskRegistryConcurrentRegistrationAndExecution(t *testing.T) {
	registry := NewTaskRegistry()
	started := make(chan struct{})
	finish := make(chan struct{})
	if err := registry.Register(Task{ID: "first", Execute: func(context.Context) (TaskResult, error) { close(started); <-finish; return TaskResult{OK: true}, nil }}); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := registry.ExecuteAll(context.Background()); done <- err }()
	<-started
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprint(i)
			_ = registry.Register(Task{ID: id, Execute: func(context.Context) (TaskResult, error) { return TaskResult{OK: true}, nil }})
			_ = registry.IDs()
			_ = registry.Status("first")
			_, _ = registry.Result("first")
		}(i)
	}
	wg.Wait()
	close(finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	results, err := registry.ExecuteAll(context.Background())
	if err != nil || len(results) != 21 {
		t.Fatal(results, err)
	}
}

func TestTaskRegistryLogging(t *testing.T) {
	for _, debug := range []bool{false, true} {
		t.Run(fmt.Sprint(debug), func(t *testing.T) {
			value := ""
			if debug {
				value = "1"
			}
			t.Setenv("GENESIS_DEBUG", value)
			var output bytes.Buffer
			registry := NewTaskRegistry(log.New(&output, "", 0))
			success := Task{ID: "success", Description: "Install prerequisite", Execute: func(context.Context) (TaskResult, error) { return TaskResult{OK: true, Details: "ready"}, nil }}
			_ = registry.Register(success)
			_ = registry.Register(success)
			_ = registry.Register(Task{ID: "bad", Description: "Failed prerequisite", Execute: func(context.Context) (TaskResult, error) { return TaskResult{Error: "permission denied"}, nil }})
			_ = registry.Register(Task{ID: "child", DependsOn: []string{"bad"}, Execute: func(context.Context) (TaskResult, error) { t.Fatal("failed dependency ran"); return TaskResult{}, nil }})
			results, err := registry.ExecuteAll(context.Background())
			if err != nil || len(results) != 3 {
				t.Fatal(results, err)
			}
			text := output.String()
			for _, expected := range []string{"Executing 3 system task(s)", "Task bad failed: permission denied", "Skipping task child due to failed dependencies", "System tasks completed: 1 succeeded, 2 failed"} {
				if !strings.Contains(text, expected) {
					t.Fatalf("missing %q in %q", expected, text)
				}
			}
			for _, expected := range []string{"Registering task: success - Install prerequisite", "Task success already registered, skipping", "Executing task: success - Install prerequisite", "Task success completed successfully: ready"} {
				if strings.Contains(text, expected) != debug {
					t.Fatalf("debug=%v, unexpected log %q: %q", debug, expected, text)
				}
			}
		})
	}
}
func TestRuntimeTaskLoggerAndReentrantStatus(t *testing.T) {
	var output bytes.Buffer
	var registry *TaskRegistry
	writer := taskLogWriter{write: func(p []byte) (int, error) {
		if registry != nil {
			_ = registry.Status("check")
			_ = registry.IDs()
		}
		return output.Write(p)
	}}
	r := &Runtime{Logger: log.New(writer, "", 0)}
	r.Ensure()
	registry = r.Tasks
	_ = registry.Register(Task{ID: "check", Execute: func(context.Context) (TaskResult, error) { return TaskResult{OK: true}, nil }})
	if _, err := registry.ExecuteAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "System tasks completed: 1 succeeded, 0 failed") {
		t.Fatal("runtime logger missing", output.String())
	}
}

type taskLogWriter struct{ write func([]byte) (int, error) }

func (w taskLogWriter) Write(p []byte) (int, error) { return w.write(p) }
func TestLastTaskCancellationPreservesContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := NewTaskRegistry()
	_ = registry.Register(Task{ID: "last", Execute: func(context.Context) (TaskResult, error) { cancel(); return TaskResult{OK: true}, nil }})
	results, err := registry.ExecuteAll(ctx)
	if !errors.Is(err, context.Canceled) || results["last"].OK || registry.Status("last") != "failed" {
		t.Fatal(results, err, registry.Status("last"))
	}
	empty := NewTaskRegistry()
	if _, err := empty.ExecuteAll(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("empty registry ignored cancellation", err)
	}
}
