package core

import (
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/metrics"
	"sort"
	"strings"
	"sync"
	"time"
)

type ExecutionPhase struct {
	ID                string        `json:"id"`
	Plugins           []Node        `json:"-"`
	Dependencies      []string      `json:"dependencies"`
	EstimatedDuration time.Duration `json:"estimatedDuration"`
	Parallelizable    bool          `json:"parallelizable"`
}
type ParallelExecutionPlan struct {
	mu            sync.Mutex
	Phases        []ExecutionPhase `json:"phases"`
	TotalPlugins  int              `json:"totalPlugins"`
	EstimatedTime time.Duration    `json:"estimatedTime"`
	Parallelism   int              `json:"parallelism"`
	elapsed       time.Duration
}
type PluginMetrics struct {
	ID                       string        `json:"id"`
	ExecutionTime            time.Duration `json:"executionTime"`
	MemoryUsage              uint64        `json:"memoryUsage"`
	CPUUsage                 float64       `json:"cpuUsage"`
	Dependencies, Dependents []string
	CriticalPath             bool `json:"criticalPath"`
}
type ExecutionResult struct {
	DidChange bool          `json:"didChange"`
	PluginID  string        `json:"pluginId"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
	Metrics   PluginMetrics `json:"metrics"`
}
type OptimizationReport struct {
	OriginalTime, OptimizedTime time.Duration
	Speedup, Efficiency         float64
	Recommendations             []string
}
type ParallelExecutionEngine struct {
	maxConcurrency int
	mu             sync.RWMutex
	metrics        map[string]PluginMetrics
	history        []ExecutionResult
}

func NewParallelExecutionEngine(maxConcurrency int) (*ParallelExecutionEngine, error) {
	if maxConcurrency < 1 {
		return nil, fmt.Errorf("maxConcurrency must be a positive integer")
	}
	return &ParallelExecutionEngine{maxConcurrency: min(maxConcurrency, max(1, runtime.GOMAXPROCS(0))), metrics: map[string]PluginMetrics{}}, nil
}
func (e *ParallelExecutionEngine) CreateExecutionPlan(nodes []Node) (*ParallelExecutionPlan, error) {
	ordered, err := BuildPluginGraph(nodes)
	if err != nil {
		return nil, err
	}
	plan := &ParallelExecutionPlan{TotalPlugins: len(nodes), Phases: []ExecutionPhase{}}
	levels := map[string]int{}
	for _, n := range ordered {
		level := 0
		for _, dep := range n.Plugin.DependsOn {
			level = max(level, levels[dep]+1)
		}
		levels[n.Instance.ID] = level
		for len(plan.Phases) <= level {
			plan.Phases = append(plan.Phases, ExecutionPhase{ID: fmt.Sprintf("phase-%d", len(plan.Phases))})
		}
		plan.Phases[level].Plugins = append(plan.Phases[level].Plugins, n)
	}
	for i := range plan.Phases {
		phase := &plan.Phases[i]
		deps := map[string]bool{}
		for _, n := range phase.Plugins {
			for _, dep := range n.Plugin.DependsOn {
				deps[dep] = true
			}
		}
		for dep := range deps {
			phase.Dependencies = append(phase.Dependencies, dep)
		}
		sort.Strings(phase.Dependencies)
		phase.Parallelizable = len(phase.Plugins) > 1 && !hasResourceConflicts(phase.Plugins, "")
		workers := 1
		if phase.Parallelizable {
			workers = min(e.maxConcurrency, len(phase.Plugins))
		}
		plan.Parallelism = max(plan.Parallelism, workers)
		slots := make([]time.Duration, workers)
		for _, n := range phase.Plugins {
			slot := 0
			for j := range slots {
				if slots[j] < slots[slot] {
					slot = j
				}
			}
			e.mu.RLock()
			metric, ok := e.metrics[n.Instance.ID]
			e.mu.RUnlock()
			duration := time.Second
			if ok {
				duration = metric.ExecutionTime
			}
			slots[slot] += duration
		}
		for _, duration := range slots {
			phase.EstimatedDuration = max(phase.EstimatedDuration, duration)
		}
		plan.EstimatedTime += phase.EstimatedDuration
	}
	return plan, nil
}
func (e *ParallelExecutionEngine) ExecuteParallel(plan *ParallelExecutionPlan, r *Runtime) ([]ExecutionResult, error) {
	r.Ensure()
	nodes := []Node{}
	for _, phase := range plan.Phases {
		nodes = append(nodes, phase.Plugins...)
	}
	if err := RunApplyPrerequisites(nodes, r); err != nil {
		return nil, err
	}
	start := time.Now()
	outcomes := map[string]ExecutionResult{}
	dependents := map[string][]string{}
	for _, n := range nodes {
		for _, dep := range n.Plugin.DependsOn {
			dependents[dep] = append(dependents[dep], n.Instance.ID)
		}
	}
	results := make([]ExecutionResult, 0, len(nodes))
	for _, phase := range plan.Phases {
		if err := r.Context.Err(); err != nil {
			return results, err
		}
		phaseResults := make([]ExecutionResult, len(phase.Plugins))
		workers := 1
		if phase.Parallelizable && !hasResourceConflicts(phase.Plugins, r.Cwd) {
			workers = min(e.maxConcurrency, len(phase.Plugins))
		}
		if workers == 1 {
			for index, n := range phase.Plugins {
				phaseResults[index] = e.executeWithDependencies(n, r, dependents, outcomes)
			}
		} else {
			jobs := make(chan int)
			var wg sync.WaitGroup
			for i := 0; i < workers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for index := range jobs {
						phaseResults[index] = e.executeWithDependencies(phase.Plugins[index], r, dependents, outcomes)
					}
				}()
			}
			for index := range phase.Plugins {
				jobs <- index
			}
			close(jobs)
			wg.Wait()
		}
		for _, result := range phaseResults {
			outcomes[result.PluginID] = result
		}
		results = append(results, phaseResults...)
	}
	plan.mu.Lock()
	plan.elapsed = time.Since(start)
	plan.mu.Unlock()
	e.mu.Lock()
	for _, result := range results {
		copy := result
		copy.Metrics = cloneMetrics(result.Metrics)
		e.history = append(e.history, copy)
	}
	e.mu.Unlock()
	return results, r.Context.Err()
}
func (e *ParallelExecutionEngine) executeWithDependencies(n Node, r *Runtime, dependents map[string][]string, outcomes map[string]ExecutionResult) ExecutionResult {
	failed := []string{}
	for _, dependency := range n.Plugin.DependsOn {
		if !outcomes[dependency].Success {
			failed = append(failed, dependency)
		}
	}
	return e.executeNode(n, r, dependents[n.Instance.ID], failed)
}

func (e *ParallelExecutionEngine) executeNode(n Node, r *Runtime, dependents, failed []string) (result ExecutionResult) {
	start := time.Now()
	heapBefore, cpuBefore := processUsage()
	result.PluginID = n.Instance.ID
	defer func() {
		if p := recover(); p != nil {
			result.Success = false
			result.Error = fmt.Sprintf("plugin panic: %v", p)
		}
		result.Duration = time.Since(start)
		heapAfter, cpuAfter := processUsage()
		memory := uint64(0)
		if heapAfter > heapBefore {
			memory = heapAfter - heapBefore
		}
		cpu := 0.0
		if elapsed := result.Duration.Seconds(); elapsed > 0 && cpuAfter > cpuBefore {
			cpu = (cpuAfter - cpuBefore) / elapsed
		}
		result.Metrics = PluginMetrics{MemoryUsage: memory, CPUUsage: cpu, ID: n.Instance.ID, ExecutionTime: result.Duration, Dependencies: append([]string(nil), n.Plugin.DependsOn...), Dependents: append([]string(nil), dependents...), CriticalPath: len(dependents) > 0}
		e.mu.Lock()
		e.metrics[n.Instance.ID] = cloneMetrics(result.Metrics)
		e.mu.Unlock()
	}()
	if len(failed) > 0 {
		result.Error = "Dependencies failed: " + strings.Join(failed, ", ")
		return
	}
	out, err := applyNode(n, r)
	result.DidChange = out.DidChange
	if err != nil {
		result.Error = err.Error()
	} else if !out.OK {
		result.Error = out.Details
		if result.Error == "" {
			result.Error = "Plugin apply failed"
		}
	} else {
		result.Success = true
	}
	return
}
func (e *ParallelExecutionEngine) GetMetrics() []PluginMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]PluginMetrics, 0, len(e.metrics))
	for _, m := range e.metrics {
		out = append(out, cloneMetrics(m))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (e *ParallelExecutionEngine) GetExecutionHistory() []ExecutionResult {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := append([]ExecutionResult(nil), e.history...)
	for i := range out {
		out[i].Metrics = cloneMetrics(out[i].Metrics)
	}
	return out
}
func (e *ParallelExecutionEngine) AnalyzePerformance(plan *ParallelExecutionPlan, results []ExecutionResult) OptimizationReport {
	plan.mu.Lock()
	elapsed := plan.elapsed
	plan.mu.Unlock()
	report := OptimizationReport{OptimizedTime: elapsed, Speedup: 1, Recommendations: []string{}}
	for _, result := range results {
		report.OriginalTime += result.Duration
		if !result.Success {
			report.Recommendations = append(report.Recommendations, "Resolve failed plugin: "+result.PluginID)
		}
		if result.Duration > 5*time.Second {
			report.Recommendations = append(report.Recommendations, "Review slow plugin: "+result.PluginID)
		}
	}
	if report.OptimizedTime > 0 {
		report.Speedup = float64(report.OriginalTime) / float64(report.OptimizedTime)
	}
	report.Efficiency = report.Speedup / float64(max(1, plan.Parallelism))
	return report
}
func (e *ParallelExecutionEngine) OptimizeExecution(nodes []Node) (*ParallelExecutionPlan, float64, error) {
	plan, err := e.CreateExecutionPlan(nodes)
	if err != nil {
		return nil, 0, err
	}
	sequential := time.Duration(0)
	for _, phase := range plan.Phases {
		for _, n := range phase.Plugins {
			e.mu.RLock()
			metric, ok := e.metrics[n.Instance.ID]
			e.mu.RUnlock()
			if ok {
				sequential += metric.ExecutionTime
			} else {
				sequential += time.Second
			}
		}
	}
	improvement := 1.0
	if plan.EstimatedTime > 0 {
		improvement = float64(sequential) / float64(plan.EstimatedTime)
	}
	return plan, improvement, nil
}
func hasResourceConflicts(nodes []Node, cwd string) bool {
	paths := []string{}
	ports := map[string]bool{}
	for _, n := range nodes {
		current := []string{}
		for _, key := range []string{"path", "install_dir", "home", "prefix", "target", "destination", "output_dir", "cache_dir"} {
			if value, ok := n.Instance.Options[key].(string); ok {
				if !filepath.IsAbs(value) {
					value = filepath.Join(cwd, value)
				}
				value, _ = filepath.Abs(value)
				for _, existing := range paths {
					if containsPath(value, existing) || containsPath(existing, value) {
						return true
					}
				}
				current = append(current, value)
			}
		}
		paths = append(paths, current...)
		if port, ok := n.Instance.Options["port"]; ok {
			key := fmt.Sprint(port)
			if ports[key] {
				return true
			}
			ports[key] = true
		}
	}
	return false
}
func containsPath(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && (relative == "." || (!filepath.IsAbs(relative) && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))))
}

func processUsage() (uint64, float64) {
	samples := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}, {Name: "/cpu/classes/user:cpu-seconds"}}
	metrics.Read(samples)
	var heap uint64
	var cpu float64
	if samples[0].Value.Kind() == metrics.KindUint64 {
		heap = samples[0].Value.Uint64()
	}
	if samples[1].Value.Kind() == metrics.KindFloat64 {
		cpu = samples[1].Value.Float64()
	}
	return heap, cpu
}
func cloneMetrics(m PluginMetrics) PluginMetrics {
	m.Dependencies = append([]string(nil), m.Dependencies...)
	m.Dependents = append([]string(nil), m.Dependents...)
	return m
}

// ClearHistory releases retained per-run results; learned duration estimates remain.
func (e *ParallelExecutionEngine) ClearHistory() { e.mu.Lock(); defer e.mu.Unlock(); e.history = nil }
func (e *ParallelExecutionEngine) ResetMetrics() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.metrics = map[string]PluginMetrics{}
}
