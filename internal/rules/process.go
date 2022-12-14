package rules

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"taskgraph/internal/execext"
	"taskgraph/internal/pm"
)

type Process struct {
	IID   string
	Deps  []string
	Cmds  []string
	Ready string

	Cwd    string
	Stdout io.Writer
	Stderr io.Writer
}

// Dependencies implements Rule
func (p *Process) Dependencies() []string {
	return p.Deps
}

// Execute implements Rule
func (p *Process) Execute(ctx context.Context) error {
	done := make(chan bool, 1)

	processManager := ctx.Value("pm.ProcessManager").(pm.ProcessManager)

	processManager.Start(func(ctx context.Context) error {
		pr, w := io.Pipe()
		r := io.TeeReader(pr, p.Stdout)

		// TODO: need to implement a timeout mechanic
		// TODO: need to support context cancellation for ctrl+c handling
		// TODO: need to be able to signal "done" with an error (i.e. timeout or ctrl+c)
		go func() {
			if p.Ready != "" {
				done <- <-WaitForText(ctx, r, p.Ready)
			} else {
				done <- true
				// TODO: this is a hack because we need to consume the reader
				// otherwise the program blocks.
				// WiatForText() consumes the reader in a goroutine which is hacky too.
				io.Copy(ioutil.Discard, r)
			}
		}()

		return execext.RunCommands(ctx, p.Cmds, &execext.RunCommandOptions{
			Env:    os.Environ(),
			Dir:    p.Cwd,
			Stdout: w,
			Stderr: p.Stderr,
		})
	})

	<-done

	return nil
}

// ID implements Rule
func (p *Process) ID() string {
	return p.IID
}

// Inputs implements Rule
func (p *Process) Inputs() []string {
	return []string{}
}

// Outputs implements Rule
func (p *Process) Outputs() []string {
	return []string{}
}

func (t *Process) Getwd() string {
	return t.Cwd
}

var _ Rule = &Process{}

// TODO: need to support context cancellation.
func WaitForText(ctx context.Context, in io.Reader, search string) chan bool {
	resultchan := make(chan bool, 1)

	scanner := bufio.NewScanner(in)

	scanner.Buffer(make([]byte, len(search)), len(search))
	scanner.Split(SplitAt(search))

	go func() {
		func() {
			for scanner.Scan() {
				resultchan <- true
				return
			}

			resultchan <- false
		}()
		io.Copy(ioutil.Discard, in)
	}()

	return resultchan
}

func SplitAt(substring string) func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, io.EOF
		}

		if i := strings.Index(string(data), substring); i >= 0 {
			return i + len(substring), data[0:i], nil
		}

		if atEOF {
			return 0, nil, io.EOF
		}

		return 1, nil, nil
	}
}
