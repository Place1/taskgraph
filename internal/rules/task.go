package rules

import (
	"context"
	"io"
	"os"
	"taskgraph/internal/execext"
)

type Task struct {
	IID  string
	Srcs []string
	Outs []string
	Deps []string
	Cmds []string

	Cwd    string
	Stdout io.Writer
	Stderr io.Writer
}

// Execute implements Rule
func (t *Task) Execute(ctx context.Context) error {
	return execext.RunCommands(ctx, t.Cmds, &execext.RunCommandOptions{
		Env:    os.Environ(),
		Dir:    t.Cwd,
		Stdout: t.Stdout,
		Stderr: t.Stderr,
	})
}

// Dependencies implements Rule
func (t *Task) Dependencies() []string {
	return t.Deps
}

// Inputs implements Rule
func (t *Task) Inputs() []string {
	return t.Srcs
}

// Outputs implements Rule
func (t *Task) Outputs() []string {
	return t.Outs
}

func (t *Task) ID() string {
	return t.IID
}

func (t *Task) Getwd() string {
	return t.Cwd
}

var _ Rule = &Task{}
