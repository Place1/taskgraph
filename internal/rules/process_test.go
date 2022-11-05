package rules

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWaitForTextMatch(t *testing.T) {
	require := require.New(t)

	r := strings.NewReader("here's my example input text")
	c := WaitForText(context.Background(), r, "my example")

	require.True(<-c)
}

func TestWaitForTextNoMatch(t *testing.T) {
	require := require.New(t)

	r := strings.NewReader("here's my example input text")
	c := WaitForText(context.Background(), r, "doesn't exist")

	require.False(<-c)
}
