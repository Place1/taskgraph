package execgraph

import (
	"context"
	"sync"
	"taskgraph/internal/execgraph/future"

	"github.com/samber/lo"
	"go.uber.org/multierr"
)

type executionNode struct {
	id    string
	graph *ExecutionGraph
	fn    func(ctx context.Context) error

	futurerw sync.RWMutex
	future   future.Future[error]
}

func (node *executionNode) execute(ctx context.Context) future.Future[error] {
	node.futurerw.Lock()
	defer node.futurerw.Unlock()

	if node.future == nil {
		node.future = future.New(func() error {
			deps := node.graph.findDependencies(node.id)

			errors := future.All(lo.Map(deps, func(dep *executionNode, i int) future.Future[error] {
				return dep.execute(ctx)
			})).Get()

			if err := multierr.Combine(errors...); err != nil {
				return err
			}

			return node.fn(ctx)
		})
	}

	return node.future
}
