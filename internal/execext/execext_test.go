package execext

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContextCancel(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool)

	go func() {
		err := RunCommand(ctx, "sleep 3", &RunCommandOptions{})
		require.ErrorIs(err, context.Canceled)
		done <- true
	}()

	<-time.After(500 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		require.Fail("command didn't exist after context cancellation")
	}
}
