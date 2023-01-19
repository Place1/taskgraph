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
	"taskgraph/internal/output"
	"taskgraph/internal/pm"
	"taskgraph/internal/rules"
	"taskgraph/internal/taskengine"
	"taskgraph/internal/taskgraph"
	"taskgraph/internal/workspace"

	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var cwd, _ = os.Getwd()

var (
	app              = kingpin.New(internal.AppName, "todo help text")
	verbose          = app.Flag("verbose", "enable verbose logging").Bool()
	workspaceDirFlag = app.Flag("workspace", "the path to the workspace directory").Default(cwd).String()

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
	processManager := pm.New(ctx)
	ctx = context.WithValue(ctx, "pm.ProcessManager", processManager)

	out := output.NewStd()
	ctx = context.WithValue(ctx, "output.OutputFactory", out)

	workspaceFile, err := findWorkspaceFile(workspaceDir)
	if err != nil {
		return err
	}

	w, err := loadWorkspace(ctx, workspaceDir)
	if err != nil {
		return err
	}

	for i := 0; i < len(w); i++ {
		w[i] = &rules.Checksum{
			Inner:        w[i],
			WorkspaceDir: filepath.Dir(workspaceFile),
			Stdout:       out.Stdout(w[i].ID()),
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

	// hack so that you can run "taskgraph run .:target" as a shorthand
	// for "//current/package:target"
	if strings.HasPrefix(target, ".:") {
		pkg, err := filepath.Rel(filepath.Dir(workspaceFile), cwd)
		if err != nil {
			return err
		}
		target = strings.Replace(target, ".", "//"+pkg, 1)
	}

	// hack so that you can run "taskgraph run :target" in a workspace
	// to run all matching targets from all packages
	if strings.HasPrefix(target, ":") {
		iid := "//-"
		g.AddTask(&rules.Task{
			IID:    iid,
			Cmds:   []string{},
			Stdout: out.Stdout(iid),
			Stderr: out.Stderr(iid),
		})
		for _, r := range w {
			if strings.HasSuffix(r.ID(), target) {
				g.AddDependency(iid, r.ID())
			}
		}
		target = iid
	}

	// hack so that you can skip typing the "//" id prefix
	if !strings.HasPrefix(target, "//") {
		target = "//" + target
	}

	engine := taskengine.New()

	logrus.Info("original tree")
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
	w, err := findWorkspaceFile(workspaceDir)
	if err != nil {
		return nil, err
	}

	r, err := workspace.Load(ctx, w)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func findWorkspaceFile(workspaceDir string) (string, error) {
	if workspaceDir == "" {
		workspaceDir = cwd
	}

	if !path.IsAbs(workspaceDir) {
		workspaceDir = filepath.Clean(filepath.Join(cwd, workspaceDir))
	}

	for {
		p := filepath.Join(workspaceDir, internal.WorkspaceFile)

		fi, err := os.Stat(p)
		if err == nil && !fi.IsDir() {
			return p, nil
		}

		if workspaceDir == filepath.Dir(workspaceDir) {
			return "", fmt.Errorf("unable to find %s file in the current directory or any parent directory", internal.WorkspaceFile)
		}

		workspaceDir = filepath.Dir(workspaceDir)
	}
}
