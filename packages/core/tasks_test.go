package core

import (
	"context"
	"errors"
	"reflect"
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
