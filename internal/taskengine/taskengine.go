package taskengine

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"taskgraph/internal/rules"
	"taskgraph/internal/taskgraph"

	"github.com/sirupsen/logrus"
	"github.com/xlab/treeprint"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type Engine interface {
	Execute(ctx context.Context, graph taskgraph.TaskGraph, task string) error
	Tree(w io.Writer, graph taskgraph.TaskGraph, taskID string) error
}

func New() Engine {
	return &engine{}
}

type engine struct {
}

func (e *engine) Tree(w io.Writer, graph taskgraph.TaskGraph, taskID string) error {
	tree := treeprint.NewWithRoot(taskID)
	t := graph.FindTask(taskID)
	if t == nil {
		return fmt.Errorf("missing task with id %s", taskID)
	}
	if err := e.tree(graph, tree, t); err != nil {
		return err
	}
	_, err := io.WriteString(w, tree.String())
	return err
}

func (e *engine) tree(graph taskgraph.TaskGraph, tree treeprint.Tree, t rules.Rule) error {
	for _, d := range graph.FindDependencies(t.ID()) {
		node := tree.AddBranch(d.ID())
		if err := e.tree(graph, node, d); err != nil {
			return err
		}
	}
	return nil
}

// Execute implements Engine
func (e *engine) Execute(ctx context.Context, graph taskgraph.TaskGraph, taskID string) error {
	walk := graphwalk{}
	limit := semaphore.NewWeighted(int64(runtime.NumCPU()))
	return walk.concurrentWalk(ctx, limit, graph, 0, graph.FindTask(taskID), func(ctx context.Context, depth int, t rules.Rule) error {
		logrus.Debugf("executing task depth=%d id=%s", depth, t.ID())
		return t.Execute(ctx)
	})
}

type graphwalk struct {
	visited sync.Map
}

func (ge *graphwalk) concurrentWalk(ctx context.Context, concurrencyLimit *semaphore.Weighted, graph taskgraph.TaskGraph, depth int, root rules.Rule, visit func(ctx context.Context, depth int, t rules.Rule) error) error {
	seen, loaded := ge.visited.LoadOrStore(root.ID(), true)
	if seen.(bool) && loaded {
		return nil
	}

	deps := graph.FindDependencies(root.ID())
	if len(deps) > 0 {
		errg, errctx := errgroup.WithContext(ctx)
		for _, t := range graph.FindDependencies(root.ID()) {
			task := t
			errg.Go(func() error {
				return ge.concurrentWalk(errctx, concurrencyLimit, graph, depth+1, task, visit)
			})
		}

		if err := errg.Wait(); err != nil {
			return err
		}
	}

	concurrencyLimit.Acquire(ctx, 1)
	defer concurrencyLimit.Release(1)
	return visit(ctx, depth, root)
}

var _ Engine = &engine{}
