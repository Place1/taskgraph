package execgraph

import (
	"context"
	"sync"

	"github.com/dominikbraun/graph"
	"github.com/samber/lo"
)

func Hello() error {

	return nil
}

type Callback func(ctx context.Context) error

type ExecutionGraph struct {
	rw    sync.RWMutex
	graph graph.Graph[string, *executionNode]
}

func New() *ExecutionGraph {
	hasher := func(node *executionNode) string {
		return node.id
	}
	return &ExecutionGraph{
		graph: graph.New(hasher, graph.Directed(), graph.Acyclic(), graph.PreventCycles()),
	}
}

func (g *ExecutionGraph) Add(id string, fn Callback) error {
	g.rw.Lock()
	defer g.rw.Unlock()
	return g.graph.AddVertex(&executionNode{
		graph: g,
		id:    id,
		fn:    fn,
	})
}

func (g *ExecutionGraph) Execute(ctx context.Context, root string) error {
	execgraph, err := g.graph.Clone()
	if err != nil {
		return err
	}

	// TODO: the Clone() method has a bug and doesn't set this property.
	// I should open an issue on their github repo.
	execgraph.Traits().PreventCycles = g.graph.Traits().PreventCycles

	if err := graph.TransitiveReduction(execgraph); err != nil {
		return err
	}

	node, err := execgraph.Vertex(root)
	if err != nil {
		return err
	}

	return node.execute(ctx).Get()
}

func (g *ExecutionGraph) AddDependency(from string, to string) error {
	g.rw.Lock()
	defer g.rw.Unlock()
	if _, err := g.graph.Edge(from, to); err == graph.ErrEdgeNotFound {
		return g.graph.AddEdge(from, to)
	}
	return nil
}

func (g *ExecutionGraph) findDependencies(id string) []*executionNode {
	g.rw.RLock()
	defer g.rw.RUnlock()

	m, err := g.graph.AdjacencyMap()
	if err != nil {
		panic(err)
	}

	deps := m[id]

	return lo.Map(lo.Keys(deps), func(key string, i int) *executionNode {
		v, _ := g.graph.Vertex(key)
		return v
	})
}
