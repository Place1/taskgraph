package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"taskgraph/internal"
	"taskgraph/internal/pm"
	"taskgraph/internal/rules"
	"taskgraph/internal/taskengine"
	"taskgraph/internal/taskgraph"
	"taskgraph/internal/workspace"

	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app              = kingpin.New(internal.AppName, "todo help text")
	verbose          = app.Flag("verbose", "enable verbose logging").Bool()
	workspaceDirFlag = app.Flag("workspace", "the path to the workspace directory").String()

	runcmd       = app.Command("run", "run a task from a build file")
	runcmdTarget = runcmd.Arg("target", "a task name from a build file").Required().String()

	listcmd = app.Command("list", "list all available tasks")
)

func main() {
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	logrus.SetLevel(logrus.InfoLevel)
	if *verbose == true {
		logrus.SetLevel(logrus.DebugLevel)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		<-ctx.Done()
		stop()
		logrus.Info("exiting...")
	}()

	var err error
	switch cmd {
	case runcmd.FullCommand():
		err = run(ctx, *runcmdTarget, *workspaceDirFlag)
	case listcmd.FullCommand():
		err = list(ctx, *workspaceDirFlag)
	default:
		logrus.Fatal(app.Help)
	}
	if err != nil {
		logrus.Fatal(err)
	}
}

func run(ctx context.Context, target string, workspaceDir string) error {
	w, err := loadWorkspace(ctx, workspaceDir)
	if err != nil {
		return err
	}

	for i := 0; i < len(w); i++ {
		w[i] = &rules.Checksum{
			Inner: w[i],
		}
	}

	g := taskgraph.New()

	for _, r := range w {
		if err := g.AddTask(r); err != nil {
			return err
		}
	}

	for _, r := range w {
		for _, d := range r.Dependencies() {
			if err := g.AddDependency(r.ID(), d); err != nil {
				return err
			}
		}
	}

	// hack so that you can run "taskgraph run :target" in a workspace
	// to run all matching targets from all packages
	if strings.HasPrefix(target, ":") {
		iid := "-"
		g.AddTask(&rules.Task{
			IID:    iid,
			Cmds:   []string{},
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		})
		for _, r := range w {
			if strings.HasSuffix(r.ID(), target) {
				g.AddDependency(iid, r.ID())
			}
		}
		target = iid
	}

	processManager := pm.New(ctx)

	ctx = context.WithValue(ctx, "pm.ProcessManager", processManager)

	engine := taskengine.New()

	if err := engine.Tree(os.Stdout, g, target); err != nil {
		return err
	}

	if err := engine.Execute(ctx, g, target); err != nil {
		return err
	}

	return processManager.Wait()
}

func list(ctx context.Context, workspaceDir string) error {
	w, err := loadWorkspace(ctx, workspaceDir)
	if err != nil {
		return err
	}

	for _, r := range w {
		fmt.Println(r.ID())
	}

	return nil
}

func loadWorkspace(ctx context.Context, workspaceDir string) ([]rules.Rule, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if workspaceDir == "" {
		workspaceDir = cwd
	}

	if !path.IsAbs(workspaceDir) {
		workspaceDir = filepath.Clean(filepath.Join(cwd, workspaceDir))
	}

	w, err := findWorkspace(workspaceDir)
	if err != nil {
		return nil, err
	}

	r, err := workspace.Load(ctx, w)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func findWorkspace(cwd string) (string, error) {
	for {
		p := filepath.Join(cwd, internal.WorkspaceFile)

		fi, err := os.Stat(p)
		if err == nil && !fi.IsDir() {
			return p, nil
		}

		if cwd == filepath.Dir(cwd) {
			return "", fmt.Errorf("unable to find %s file in the current directory or any parent directory", internal.WorkspaceFile)
		}

		cwd = filepath.Dir(cwd)
	}
}
