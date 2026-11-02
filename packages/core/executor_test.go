package core

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func testNode(id string, depends ...string) Node {
	return Node{Instance: Instance{ID: id, Category: "tool", Module: id}, Plugin: Plugin{ID: id, Category: "tool", DependsOn: depends}}
}
func TestPluginGraph(t *testing.T) {
	nodes := []Node{testNode("last", "first"), testNode("first")}
	ordered, err := BuildPluginGraph(nodes)
	if err != nil || ordered[0].Instance.ID != "first" {
		t.Fatal(ordered, err)
	}
	for _, nodes := range [][]Node{{testNode("a"), testNode("a")}, {testNode("a", "missing")}, {testNode("a", "b"), testNode("b", "a")}} {
		if _, err := BuildPluginGraph(nodes); err == nil {
			t.Fatal("invalid graph accepted")
		}
	}
}
func TestPluginLifecycle(t *testing.T) {
	events := []string{}
	nodes := []Node{testNode("first"), testNode("last", "first")}
	for i := range nodes {
		id := nodes[i].Instance.ID
		p := &nodes[i].Plugin
		p.Prepare = func(*Runtime, Instance) error { events = append(events, "prepare:"+id); return nil }
		p.RegisterTasks = func(r *Runtime, _ Instance) error {
			events = append(events, "register:"+id)
			return r.Tasks.Register(Task{ID: "shared", Execute: func(context.Context) (TaskResult, error) {
				events = append(events, "task")
				return TaskResult{OK: true}, nil
			}})
		}
		p.PreApply = func(*Runtime, Instance) error { events = append(events, "pre:"+id); return nil }
		p.Apply = func(*Runtime, Instance) (ApplyResult, error) {
			events = append(events, "apply:"+id)
			return ApplyResult{OK: true, DidChange: true}, nil
		}
		p.PostApply = func(*Runtime, Instance) error { events = append(events, "post:"+id); return nil }
	}
	results, err := RunApply(nodes, &Runtime{})
	if err != nil || len(results) != 2 || !results[0].DidChange {
		t.Fatal(results, err)
	}
	expected := []string{"prepare:first", "prepare:last", "register:first", "register:last", "task", "pre:first", "apply:first", "post:first", "pre:last", "apply:last", "post:last"}
	if !reflect.DeepEqual(events, expected) {
		t.Fatal(events)
	}
}
func TestPluginFailurePropagation(t *testing.T) {
	nodes := []Node{testNode("bad"), testNode("child", "bad"), testNode("unrelated")}
	nodes[0].Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) { return ApplyResult{Details: "failed"}, nil }
	nodes[0].Plugin.PostApply = func(*Runtime, Instance) error { t.Fatal("post on failure"); return nil }
	nodes[1].Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) { t.Fatal("dependent ran"); return ApplyResult{}, nil }
	nodes[2].Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) { return ApplyResult{OK: true}, nil }
	results, err := RunApply(nodes, &Runtime{})
	if err != nil || results[0].OK || results[1].OK || !results[2].OK {
		t.Fatal(results, err)
	}
}
func TestPluginTaskFailureStopsApply(t *testing.T) {
	node := testNode("one")
	node.Plugin.RegisterTasks = func(r *Runtime, _ Instance) error {
		return r.Tasks.Register(Task{ID: "broken", Execute: func(context.Context) (TaskResult, error) { return TaskResult{Error: "failed"}, nil }})
	}
	node.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) {
		t.Fatal("applied after failed prerequisite")
		return ApplyResult{}, nil
	}
	if _, err := RunApply([]Node{node}, &Runtime{}); err == nil || !strings.Contains(err.Error(), "broken") {
		t.Fatal(err)
	}
}
func TestPluginThrownErrors(t *testing.T) {
	for _, stage := range []string{"prepare", "register", "pre", "apply", "post", "detect", "validate"} {
		t.Run(stage, func(t *testing.T) {
			node := testNode("one")
			failure := errors.New("failure")
			hook := func(*Runtime, Instance) error { return failure }
			switch stage {
			case "prepare":
				node.Plugin.Prepare = hook
			case "register":
				node.Plugin.RegisterTasks = hook
			case "pre":
				node.Plugin.PreApply = hook
			case "apply":
				node.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) { return ApplyResult{}, failure }
			case "post":
				node.Plugin.PostApply = hook
			case "detect":
				node.Plugin.Detect = func(*Runtime, Instance) (DetectResult, error) { return DetectResult{}, failure }
			case "validate":
				node.Plugin.Validate = func(*Runtime, Instance) (ValidateResult, error) { return ValidateResult{}, failure }
			}
			var err error
			switch stage {
			case "detect":
				_, err = RunDetect([]Node{node}, &Runtime{})
			case "validate":
				_, err = RunValidate([]Node{node}, &Runtime{})
			default:
				_, err = RunApply([]Node{node}, &Runtime{})
			}
			if !errors.Is(err, failure) {
				t.Fatal(err)
			}
		})
	}
}
func TestDetectValidateMissingHooks(t *testing.T) {
	nodes := []Node{testNode("nohooks"), testNode("present"), testNode("missing")}
	nodes[1].Plugin.Detect = func(*Runtime, Instance) (DetectResult, error) { return DetectResult{OK: true, Details: "yes"}, nil }
	nodes[2].Plugin.Detect = func(*Runtime, Instance) (DetectResult, error) { return DetectResult{Details: "no"}, nil }
	nodes[2].Plugin.Validate = func(*Runtime, Instance) (ValidateResult, error) { return ValidateResult{Message: "no"}, nil }
	detected, err := RunDiff(nodes, &Runtime{})
	if err != nil || detected[0].Status != "unknown" || detected[1].Status != "present" || detected[2].Status != "missing" {
		t.Fatal(detected, err)
	}
	validated, err := RunValidate(nodes, &Runtime{})
	if err != nil || !validated[0].OK || validated[2].OK {
		t.Fatal(validated, err)
	}
}

func TestGraphTraversalPreservesStableOrderAndInput(t *testing.T) {
	nodes := []Node{testNode("last", "left", "right"), testNode("right", "base"), testNode("left", "base"), testNode("base"), testNode("independent")}
	expected := []string{"base", "left", "right", "last", "independent"}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ordered, err := BuildPluginGraph(nodes)
			if err != nil {
				t.Error(err)
				return
			}
			ids := make([]string, len(ordered))
			for i, n := range ordered {
				ids[i] = n.Instance.ID
			}
			if !reflect.DeepEqual(ids, expected) {
				t.Error(ids)
			}
			ordered[0].Instance.ID = "changed"
		}()
	}
	wg.Wait()
	if nodes[0].Instance.ID != "last" || nodes[3].Instance.ID != "base" {
		t.Fatal("input order or identity mutated", nodes)
	}
}
func TestPrerequisitesRejectInvalidGraphBeforeHooks(t *testing.T) {
	node := testNode("bad", "missing")
	node.Plugin.Prepare = func(*Runtime, Instance) error { t.Fatal("invalid graph reached hook"); return nil }
	if err := RunApplyPrerequisites([]Node{node}, &Runtime{}); err == nil {
		t.Fatal("missing dependency accepted")
	}
}
func TestLastSequentialPluginCancellationPreservesContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	node := testNode("last")
	node.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) { cancel(); return ApplyResult{OK: true}, nil }
	results, err := RunApply([]Node{node}, &Runtime{Context: ctx})
	if !errors.Is(err, context.Canceled) || len(results) != 1 {
		t.Fatal(results, err)
	}
}

func TestApplyErrorsRetainCurrentPartialSummary(t *testing.T) {
	for _, stage := range []string{"pre", "apply", "post"} {
		t.Run(stage, func(t *testing.T) {
			failure := errors.New("operation failed")
			previous, current, later := testNode("previous"), testNode("current"), testNode("later")
			previous.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) {
				return ApplyResult{OK: true, DidChange: true, Details: "previous finished"}, nil
			}
			current.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) {
				return ApplyResult{OK: true, DidChange: true, Details: "installation changed"}, nil
			}
			switch stage {
			case "pre":
				current.Plugin.PreApply = func(*Runtime, Instance) error { return failure }
				current.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) {
					t.Fatal("apply after pre hook error")
					return ApplyResult{}, nil
				}
			case "apply":
				current.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) {
					return ApplyResult{DidChange: true, Details: "partial installation"}, failure
				}
			case "post":
				current.Plugin.PostApply = func(*Runtime, Instance) error { return failure }
			}
			later.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) {
				t.Fatal("plugin ran after error")
				return ApplyResult{}, nil
			}
			summaries, err := RunApply([]Node{previous, current, later}, &Runtime{})
			if !errors.Is(err, failure) || len(summaries) != 2 {
				t.Fatal(summaries, err)
			}
			if !summaries[0].OK || !summaries[0].DidChange || summaries[0].Details != "previous finished" {
				t.Fatal("previous summary lost", summaries[0])
			}
			summary := summaries[1]
			if summary.ID != "current" || summary.Category != "tool" || summary.OK || summary.DidChange != (stage != "pre") || !strings.Contains(summary.Details, failure.Error()) {
				t.Fatal(summary)
			}
			if stage == "apply" && !strings.Contains(summary.Details, "partial installation") {
				t.Fatal("partial details lost", summary)
			}
		})
	}
}
