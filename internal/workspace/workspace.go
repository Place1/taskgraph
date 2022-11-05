package workspace

import (
	"context"
	"os"
	"path/filepath"
	"taskgraph/internal"
	"taskgraph/internal/rules"
	"taskgraph/internal/workspace/starbuild"
)

func Load(ctx context.Context, workspace string) ([]rules.Rule, error) {
	buildfiles := []string{}

	err := filepath.Walk(filepath.Dir(workspace), func(path string, info os.FileInfo, err error) error {
		if err == nil && info.Name() == internal.BuildFile {
			buildfiles = append(buildfiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	r := []rules.Rule{}
	for _, bf := range buildfiles {
		x, err := starbuild.Exec(ctx, packageName(workspace, bf), bf)
		if err != nil {
			return nil, err
		}
		r = append(r, x...)
	}

	return r, nil
}

func packageName(workspace string, buildfile string) string {
	x, err := filepath.Rel(filepath.Dir(workspace), filepath.Dir(buildfile))
	if err != nil {
		panic(err)
	}
	return "//" + x
}
