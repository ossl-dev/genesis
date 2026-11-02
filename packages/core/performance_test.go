package core

import (
	"fmt"
	"testing"
)

func benchmarkNodes(count int) []Node {
	nodes := make([]Node, count)
	for i := range nodes {
		nodes[i] = Node{Instance: Instance{ID: fmt.Sprintf("plugin-%d", i), Category: "tool"}}
		if i > 0 {
			nodes[i].Plugin.DependsOn = []string{nodes[i-1].Instance.ID}
		}
	}
	return nodes
}

func BenchmarkBuildPluginGraph(b *testing.B) {
	for _, count := range []int{11, 1000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			nodes := benchmarkNodes(count)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := BuildPluginGraph(nodes); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkApplyEnvironment(b *testing.B) {
	for _, count := range []int{11, 1000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			nodes := benchmarkNodes(count)
			r := &Runtime{Env: NewEnvironment(map[string]string{})}
			r.Ensure()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := ApplyEnvironment(Config{}, nodes, r); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParallelDependencyChain(b *testing.B) {
	nodes := benchmarkNodes(100)
	engine, err := NewParallelExecutionEngine(4)
	if err != nil {
		b.Fatal(err)
	}
	plan, err := engine.CreateExecutionPlan(nodes)
	if err != nil {
		b.Fatal(err)
	}
	r := &Runtime{Env: NewEnvironment(map[string]string{})}
	r.Ensure()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.ExecuteParallel(plan, r); err != nil {
			b.Fatal(err)
		}
		engine.ClearHistory()
	}
}
