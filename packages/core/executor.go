package core

import (
	"fmt"
	"strings"
)

type DetectSummary struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Details  string `json:"details,omitempty"`
}
type ApplySummary struct {
	ID        string `json:"id"`
	Category  string `json:"category"`
	OK        bool   `json:"ok"`
	DidChange bool   `json:"didChange"`
	Details   string `json:"details,omitempty"`
}
type ValidateSummary struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	OK       bool   `json:"ok"`
	Message  string `json:"message,omitempty"`
}

func BuildPluginGraph(nodes []Node) ([]Node, error) {
	byID := make(map[string]int, len(nodes))
	for index := range nodes {
		id := nodes[index].Instance.ID
		if _, exists := byID[id]; exists {
			return nil, fmt.Errorf("duplicate plugin id '%s'", id)
		}
		byID[id] = index
	}
	states := make([]uint8, len(nodes))
	ordered := make([]Node, 0, len(nodes))
	var visit func(int) error
	visit = func(index int) error {
		if states[index] == 2 {
			return nil
		}
		n := &nodes[index]
		if states[index] == 1 {
			return fmt.Errorf("cyclic plugin dependency involving '%s'", n.Instance.ID)
		}
		states[index] = 1
		for _, id := range n.Plugin.DependsOn {
			dependency, exists := byID[id]
			if !exists {
				return fmt.Errorf("missing plugin '%s' referenced in dependency", id)
			}
			if err := visit(dependency); err != nil {
				return err
			}
		}
		states[index] = 2
		ordered = append(ordered, *n)
		return nil
	}
	for index := range nodes {
		if err := visit(index); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}
func RunDetect(nodes []Node, r *Runtime) ([]DetectSummary, error) {
	r.Ensure()
	results := make([]DetectSummary, 0, len(nodes))
	for _, n := range nodes {
		if err := r.Context.Err(); err != nil {
			return results, err
		}
		summary := DetectSummary{ID: n.Instance.ID, Category: n.Instance.Category, Status: "unknown"}
		if n.Plugin.Detect != nil {
			v, err := n.Plugin.Detect(r, n.Instance)
			if err != nil {
				return results, fmt.Errorf("plugin %s detect: %w", n.Instance.ID, err)
			}
			summary.Status = "missing"
			if v.OK {
				summary.Status = "present"
			}
			summary.Details = v.Details
		}
		results = append(results, summary)
	}
	return results, nil
}
func RunDiff(nodes []Node, r *Runtime) ([]DetectSummary, error) { return RunDetect(nodes, r) }
func RunValidate(nodes []Node, r *Runtime) ([]ValidateSummary, error) {
	r.Ensure()
	results := make([]ValidateSummary, 0, len(nodes))
	for _, n := range nodes {
		if err := r.Context.Err(); err != nil {
			return results, err
		}
		summary := ValidateSummary{ID: n.Instance.ID, Category: n.Instance.Category, OK: true}
		if n.Plugin.Validate != nil {
			v, err := n.Plugin.Validate(r, n.Instance)
			if err != nil {
				return results, fmt.Errorf("plugin %s validate: %w", n.Instance.ID, err)
			}
			summary.OK = v.OK
			summary.Message = v.Message
		}
		results = append(results, summary)
	}
	return results, nil
}
func RunApplyPrerequisites(nodes []Node, r *Runtime) error {
	ordered, err := BuildPluginGraph(nodes)
	if err != nil {
		return err
	}
	return runApplyPrerequisitesOrdered(ordered, r)
}

// Internal callers may reuse a graph validated immediately before this call.
func runApplyPrerequisitesOrdered(ordered []Node, r *Runtime) error {
	r.Ensure()
	for _, hook := range []func(Plugin) Hook{func(p Plugin) Hook { return p.Prepare }, func(p Plugin) Hook { return p.RegisterTasks }} {
		for _, n := range ordered {
			if err := r.Context.Err(); err != nil {
				return err
			}
			if h := hook(n.Plugin); h != nil {
				if err := h(r, n.Instance); err != nil {
					return fmt.Errorf("plugin %s prerequisites: %w", n.Instance.ID, err)
				}
			}
		}
	}
	results, err := r.Tasks.ExecuteAll(r.Context)
	if err != nil {
		return err
	}
	failures := []string{}
	for _, id := range r.Tasks.IDs() {
		v := results[id]
		if !v.OK {
			failures = append(failures, id+": "+v.Error+v.Details)
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("system tasks failed: %s", strings.Join(failures, "; "))
	}
	return nil
}
func applyNode(n Node, r *Runtime) (ApplyResult, error) {
	if err := r.Context.Err(); err != nil {
		return ApplyResult{}, err
	}
	if n.Plugin.PreApply != nil {
		if err := n.Plugin.PreApply(r, n.Instance); err != nil {
			return ApplyResult{}, err
		}
	}
	out := ApplyResult{OK: true}
	var err error
	if n.Plugin.Apply != nil {
		out, err = n.Plugin.Apply(r, n.Instance)
		if err != nil {
			return out, err
		}
	}
	if out.OK && n.Plugin.PostApply != nil {
		err = n.Plugin.PostApply(r, n.Instance)
	}
	return out, err
}
func RunApply(nodes []Node, r *Runtime) ([]ApplySummary, error) {
	ordered, err := BuildPluginGraph(nodes)
	if err != nil {
		return nil, err
	}
	if err = runApplyPrerequisitesOrdered(ordered, r); err != nil {
		return nil, err
	}
	results := make([]ApplySummary, 0, len(nodes))
	failed := map[string]bool{}
	for _, n := range ordered {
		deps := []string{}
		for _, dep := range n.Plugin.DependsOn {
			if failed[dep] {
				deps = append(deps, dep)
			}
		}
		var out ApplyResult
		if len(deps) > 0 {
			out = ApplyResult{Details: "Dependencies failed: " + strings.Join(deps, ", ")}
		} else {
			out, err = applyNode(n, r)
			if err != nil {
				return results, fmt.Errorf("plugin %s apply: %w", n.Instance.ID, err)
			}
		}
		failed[n.Instance.ID] = !out.OK
		results = append(results, ApplySummary{ID: n.Instance.ID, Category: n.Instance.Category, OK: out.OK, DidChange: out.DidChange, Details: out.Details})
	}
	return results, r.Context.Err()
}
