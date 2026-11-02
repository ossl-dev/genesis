package core

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestParallelBoundedAndDependent(t *testing.T) {
	var current, peak atomic.Int32
	nodes := []Node{}
	for _, id := range []string{"one", "two", "three", "four"} {
		node := testNode(id)
		node.Plugin.Apply = func(r *Runtime, i Instance) (ApplyResult, error) {
			n := current.Add(1)
			for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
			}
			r.Env.Set(i.ID, "yes")
			time.Sleep(10 * time.Millisecond)
			current.Add(-1)
			return ApplyResult{OK: true, DidChange: true}, nil
		}
		nodes = append(nodes, node)
	}
	last := testNode("last", "one", "two", "three", "four")
	last.Plugin.Apply = func(r *Runtime, _ Instance) (ApplyResult, error) {
		if current.Load() != 0 || r.Env.Get("four") != "yes" {
			t.Error("dependency unfinished")
		}
		return ApplyResult{OK: true}, nil
	}
	nodes = append(nodes, last)
	engine, err := NewParallelExecutionEngine(2)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := engine.CreateExecutionPlan(nodes)
	if err != nil || len(plan.Phases) != 2 || plan.TotalPlugins != 5 {
		t.Fatal(plan, err)
	}
	results, err := engine.ExecuteParallel(plan, &Runtime{})
	if err != nil || len(results) != 5 || !results[0].DidChange || peak.Load() > 2 {
		t.Fatal(results, peak.Load(), err)
	}
	if len(engine.GetMetrics()) != 5 || len(engine.GetExecutionHistory()) != 5 {
		t.Fatal("metrics lost")
	}
	report := engine.AnalyzePerformance(plan, results)
	if report.OriginalTime <= 0 || report.OptimizedTime <= 0 || report.Speedup <= 0 {
		t.Fatal(report)
	}
	if _, _, err := engine.OptimizeExecution(nodes); err != nil {
		t.Fatal(err)
	}
}
func TestParallelConflicts(t *testing.T) {
	for _, opts := range []struct{ a, b Options }{{Options{"install_dir": "/a"}, Options{"install_dir": "/a/b"}}, {Options{"path": "relative"}, Options{"path": "relative/child"}}, {Options{"port": 8080}, Options{"port": "8080"}}} {
		a, b := testNode("a"), testNode("b")
		a.Instance.Options = opts.a
		b.Instance.Options = opts.b
		engine, _ := NewParallelExecutionEngine(4)
		plan, err := engine.CreateExecutionPlan([]Node{a, b})
		if err != nil || plan.Phases[0].Parallelizable {
			t.Fatal(plan, err)
		}
	}
	a, b := testNode("a"), testNode("b")
	a.Instance.Options = Options{"path": "/alpha"}
	b.Instance.Options = Options{"path": "/alphabet"}
	if hasResourceConflicts([]Node{a, b}, "") {
		t.Fatal("unrelated sibling paths conflict")
	}
}
func TestParallelFailureAndPanic(t *testing.T) {
	for _, panicValue := range []bool{false, true} {
		first, second, third := testNode("first"), testNode("second", "first"), testNode("third")
		first.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) {
			if panicValue {
				panic("broken")
			}
			return ApplyResult{Details: "broken"}, nil
		}
		second.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) {
			t.Fatal("failed dependent ran")
			return ApplyResult{}, nil
		}
		engine, _ := NewParallelExecutionEngine(2)
		plan, _ := engine.CreateExecutionPlan([]Node{first, second, third})
		results, err := engine.ExecuteParallel(plan, &Runtime{})
		if err != nil {
			t.Fatal(err)
		}
		byID := map[string]ExecutionResult{}
		for _, r := range results {
			byID[r.PluginID] = r
		}
		if byID["first"].Success || byID["second"].Success || !byID["third"].Success {
			t.Fatal(results)
		}
	}
}
func TestParallelInvalidConcurrencyAndCancel(t *testing.T) {
	if _, err := NewParallelExecutionEngine(0); err == nil {
		t.Fatal("invalid concurrency accepted")
	}
	engine, _ := NewParallelExecutionEngine(2)
	plan, _ := engine.CreateExecutionPlan([]Node{testNode("one")})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := engine.ExecuteParallel(plan, &Runtime{Context: ctx}); err == nil {
		t.Fatal("cancellation ignored")
	}
}
func TestEnvironmentConcurrentAccess(t *testing.T) {
	env := NewEnvironment(map[string]string{"PATH": "/a"})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			env.Set("key", "value")
			_ = env.Get("key")
			env.PrependPath("/b")
			_ = env.Snapshot()
		}()
	}
	wg.Wait()
	if env.Get("key") != "value" {
		t.Fatal("value lost")
	}
}
func TestParallelResultsDoNotAliasStoredMetrics(t *testing.T) {
	first, last := testNode("first"), testNode("last", "first")
	engine, _ := NewParallelExecutionEngine(2)
	plan, _ := engine.CreateExecutionPlan([]Node{first, last})
	results, err := engine.ExecuteParallel(plan, &Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	results[0].Metrics.Dependents[0] = "changed"
	results[1].Metrics.Dependencies[0] = "changed"
	for _, m := range engine.GetMetrics() {
		for _, id := range append(m.Dependencies, m.Dependents...) {
			if id == "changed" {
				t.Fatal("returned result aliases stored metrics")
			}
		}
	}
	history := engine.GetExecutionHistory()
	if history[0].Metrics.Dependents[0] != "last" || history[1].Metrics.Dependencies[0] != "first" {
		t.Fatal("result aliases stored history")
	}
	history[0].Metrics.Dependents[0] = "mutated"
	if engine.GetExecutionHistory()[0].Metrics.Dependents[0] != "last" {
		t.Fatal("history getter aliases internal values")
	}
	engine.ClearHistory()
	engine.ResetMetrics()
	if len(engine.GetExecutionHistory()) != 0 || len(engine.GetMetrics()) != 0 {
		t.Fatal("reset failed")
	}
}
func TestEnvironmentZeroValue(t *testing.T) {
	var e Environment
	e.Set("value", "yes")
	e.PrependPath("path")
	if e.Get("value") != "yes" {
		t.Fatal(e.Snapshot())
	}
}
func TestLastParallelPhaseCancellationPreservesContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	node := testNode("last")
	node.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) { cancel(); return ApplyResult{OK: true}, nil }
	engine, _ := NewParallelExecutionEngine(2)
	plan, _ := engine.CreateExecutionPlan([]Node{node})
	results, err := engine.ExecuteParallel(plan, &Runtime{Context: ctx})
	if !errors.Is(err, context.Canceled) || len(results) != 1 {
		t.Fatal(results, err)
	}
}
func TestSerializedResourcePhaseKeepsLifecycleAndPanicIsolation(t *testing.T) {
	events := []string{}
	first, second := testNode("first"), testNode("second")
	first.Instance.Options = Options{"install_dir": "/shared"}
	second.Instance.Options = Options{"install_dir": "/shared/child"}
	first.Plugin.PreApply = func(*Runtime, Instance) error { events = append(events, "pre:first"); return nil }
	first.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) { events = append(events, "apply:first"); panic("failed") }
	first.Plugin.PostApply = func(*Runtime, Instance) error { t.Fatal("post hook after panic"); return nil }
	second.Plugin.PreApply = func(*Runtime, Instance) error { events = append(events, "pre:second"); return nil }
	second.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) {
		events = append(events, "apply:second")
		return ApplyResult{OK: true, DidChange: true}, nil
	}
	second.Plugin.PostApply = func(*Runtime, Instance) error { events = append(events, "post:second"); return nil }
	engine, _ := NewParallelExecutionEngine(4)
	plan, err := engine.CreateExecutionPlan([]Node{first, second})
	if err != nil || plan.Phases[0].Parallelizable {
		t.Fatal(plan, err)
	}
	results, err := engine.ExecuteParallel(plan, &Runtime{})
	if err != nil || len(results) != 2 || results[0].Success || !results[1].Success || !results[1].DidChange {
		t.Fatal(results, err)
	}
	expected := []string{"pre:first", "apply:first", "pre:second", "apply:second", "post:second"}
	if !reflect.DeepEqual(events, expected) {
		t.Fatal(events)
	}
}
