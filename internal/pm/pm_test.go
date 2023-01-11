package pm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunsProcess(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := New(ctx)

	didrun := false
	p.Start(func(ctx context.Context) error {
		didrun = true
		return nil
	})

	require.NoError(p.Wait())
	require.True(didrun)
}

func TestWaitsForCancel(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithCancel(context.Background())

	p := New(ctx)

	didrun := false
	p.Start(func(ctx context.Context) error {
		// wait for cancel
		<-ctx.Done()
		// delay for a short time
		<-time.After(50 * time.Second)
		// complete
		didrun = true
		return nil
	})

	cancel()

	require.NoError(p.Wait())
	require.True(didrun)
}
