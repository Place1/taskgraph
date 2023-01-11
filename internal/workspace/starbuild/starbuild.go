package starbuild

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"taskgraph/internal/output"
	"taskgraph/internal/rules"

	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"go.starlark.net/starlark"
)

func Exec(ctx context.Context, packageName string, file string) ([]rules.Rule, error) {
	thread := &starlark.Thread{
		Name: file,
	}

	cwd := filepath.Dir(file)

	out := ctx.Value("output.OutputFactory").(output.OutputFactory)

	r := []rules.Rule{}

	task := starlark.NewBuiltin("task", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		name := ""
		srcs := &starlark.List{}
		deps := &starlark.List{}
		outs := &starlark.List{}
		cmds := &starlark.List{}
		if err := starlark.UnpackArgs(fn.Name(), args, kwargs,
			"name", &name,
			"srcs?", &srcs,
			"outs?", &outs,
			"deps?", &deps,
			"cmds", &cmds); err != nil {
			return nil, err
		}

		fqname := fmt.Sprintf("%s:%s", packageName, name)

		r = append(r, &rules.Task{
			IID:  fqname,
			Srcs: tostrarr(srcs),
			Outs: tostrarr(outs),
			Cmds: tostrarr(cmds),
			Deps: lo.Map(tostrarr(deps), func(d string, i int) string {
				if strings.HasPrefix(d, ":") {
					return fmt.Sprintf("%s%s", packageName, d)
				}
				return d
			}),

			Cwd:    cwd,
			Stdout: out.Stdout(fqname),
			Stderr: out.Stderr(fqname),
		})

		return starlark.None, nil
	})

	process := starlark.NewBuiltin("process", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		name := ""
		deps := &starlark.List{}
		cmds := &starlark.List{}
		ready := ""
		if err := starlark.UnpackArgs(fn.Name(), args, kwargs,
			"name", &name,
			"deps?", &deps,
			"cmds", &cmds,
			"ready", &ready); err != nil {
			return nil, err
		}

		fqname := fmt.Sprintf("%s:%s", packageName, name)

		r = append(r, &rules.Process{
			IID:  fqname,
			Cmds: tostrarr(cmds),
			Deps: lo.Map(tostrarr(deps), func(d string, i int) string {
				if strings.HasPrefix(d, ":") {
					return fmt.Sprintf("%s%s", packageName, d)
				}
				return d
			}),
			Ready: ready,

			Cwd:    cwd,
			Stdout: out.Stdout(fqname),
			Stderr: out.Stderr(fqname),
		})

		return starlark.None, nil
	})

	filegroup := starlark.NewBuiltin("filegroup", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		name := ""
		srcs := &starlark.List{}
		if err := starlark.UnpackArgs(fn.Name(), args, kwargs,
			"name", &name,
			"srcs", &srcs); err != nil {
			return nil, err
		}

		fqname := fmt.Sprintf("%s:%s", packageName, name)

		r = append(r, &rules.Filegroup{
			IID:  fqname,
			Srcs: tostrarr(srcs),
			Cwd:  cwd,
		})

		return starlark.None, nil
	})

	if _, err := starlark.ExecFile(thread, file, nil, starlark.StringDict{
		"task":      task,
		"process":   process,
		"filegroup": filegroup,
	}); err != nil {
		return nil, err
	}

	return r, nil
}

func tostrarr(l *starlark.List) []string {
	out := make([]string, l.Len())
	i := 0
	iter := l.Iterate()
	var v starlark.Value
	for iter.Next(&v) {
		dep, ok := starlark.AsString(v)
		if !ok {
			logrus.Warnf("non-string values will coerced to strings %s -> %s", v.Type(), v.String())
			dep = v.String()
		}
		out[i] = dep
		i++
	}
	return out
}
