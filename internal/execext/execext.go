// https://github.com/go-task/task/blob/master/internal/execext/exec.go

package execext

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"
)

// RunCommandOptions is the options for the RunCommand func
type RunCommandOptions struct {
	Dir    string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

var (
	// ErrNilOptions is returned when a nil options is given
	ErrNilOptions = errors.New("execext: nil options given")
)

// RunCommand runs a shell command
func RunCommand(ctx context.Context, cmd string, opts *RunCommandOptions) error {
	if opts == nil {
		return ErrNilOptions
	}

	p, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		return err
	}

	environ := opts.Env
	if len(environ) == 0 {
		environ = os.Environ()
	}

	r, err := interp.New(
		interp.Params("-e"),
		interp.Env(expand.ListEnviron(environ...)),
		interp.ExecHandler(ExecHandler(2*time.Second)),
		interp.OpenHandler(openHandler),
		interp.StdIO(opts.Stdin, opts.Stdout, opts.Stderr),
		dirOption(opts.Dir),
	)
	if err != nil {
		return err
	}

	return r.Run(ctx, p)
}

// RunCommand runs a list of shell commands
func RunCommands(ctx context.Context, cmds []string, opts *RunCommandOptions) error {
	for _, cmd := range cmds {
		if err := RunCommand(ctx, cmd, opts); err != nil {
			return err
		}
	}
	return nil
}

// IsExitError returns true the given error is an exis status error
func IsExitError(err error) bool {
	if _, ok := interp.IsExitStatus(err); ok {
		return true
	}
	return false
}

// Expand is a helper to mvdan.cc/shell.Fields that returns the first field
// if available.
func Expand(s string) (string, error) {
	s = filepath.ToSlash(s)
	s = strings.ReplaceAll(s, " ", `\ `)
	fields, err := shell.Fields(s, nil)
	if err != nil {
		return "", err
	}
	if len(fields) > 0 {
		return fields[0], nil
	}
	return "", nil
}

func openHandler(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	if path == "/dev/null" {
		return devNull{}, nil
	}
	return interp.DefaultOpenHandler()(ctx, path, flag, perm)
}

func dirOption(path string) interp.RunnerOption {
	return func(r *interp.Runner) error {
		err := interp.Dir(path)(r)
		if err == nil {
			return nil
		}

		// If the specified directory doesn't exist, it will be created later.
		// Therefore, even if `interp.Dir` method returns an error, the
		// directory path should be set only when the directory cannot be found.
		if absPath, _ := filepath.Abs(path); absPath != "" {
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				r.Dir = absPath
				return nil
			}
		}

		return err
	}
}

func ExecHandler(killTimeout time.Duration) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		hc := interp.HandlerCtx(ctx)
		path, err := interp.LookPathDir(hc.Dir, hc.Env, args[0])
		if err != nil {
			fmt.Fprintln(hc.Stderr, err)
			return interp.NewExitStatus(127)
		}
		cmd := exec.Cmd{
			Path: path,
			Args: args,
			// Env:    hc.Env,
			Dir:    hc.Dir,
			Stdin:  hc.Stdin,
			Stdout: hc.Stdout,
			Stderr: hc.Stderr,
		}

		wg, ctx := errgroup.WithContext(ctx)
		procdone := make(chan struct{}, 1)

		wg.Go(func() error {
			defer func() {
				procdone <- struct{}{}
			}()

			if err := cmd.Start(); err != nil {
				return err
			}

			return cmd.Wait()
		})

		wg.Go(func() error {
			done := false

			select {
			case <-ctx.Done():
			case <-procdone:
				done = true
			}

			if !done {
				if killTimeout <= 0 || runtime.GOOS == "windows" {
					return cmd.Process.Signal(os.Kill)
				}

				cmd.Process.Signal(os.Interrupt)

				select {
				case <-procdone:
					return nil
				case <-time.After(killTimeout):
					return cmd.Process.Signal(os.Kill)
				}
			} else {
				return nil
			}
		})

		err = wg.Wait()

		switch x := err.(type) {
		case *exec.ExitError:
			// started, but errored - default to 1 if OS
			// doesn't have exit statuses
			if status, ok := x.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					return interp.NewExitStatus(uint8(128 + status.Signal()))
				}
				return interp.NewExitStatus(uint8(status.ExitStatus()))
			}
			return interp.NewExitStatus(1)
		case *exec.Error:
			// did not start
			fmt.Fprintf(hc.Stderr, "%v\n", err)
			return interp.NewExitStatus(127)
		default:
			return err
		}
	}
}
