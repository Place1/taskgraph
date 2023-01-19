package execgraph

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	require := require.New(t)

	graph := New()

	printer := func(name string) Callback {
		return func(ctx context.Context) error {
			fmt.Println(name)
			return nil
		}
	}

	// https://dominikbraun.io/blog/graphs/reducing-graph-complexity-using-go-and-transitive-reduction/

	graph.Add("1", printer("1"))
	graph.Add("2", printer("2"))
	graph.Add("3", printer("3"))
	graph.Add("4", printer("4"))
	graph.Add("5", printer("5"))

	graph.AddDependency("1", "2")
	graph.AddDependency("1", "3")
	graph.AddDependency("1", "4")
	graph.AddDependency("1", "5")
	graph.AddDependency("2", "4")
	graph.AddDependency("3", "4")
	graph.AddDependency("3", "5")
	graph.AddDependency("4", "5")

	err := graph.Execute(context.Background(), "1")
	require.NoError(err)
}
