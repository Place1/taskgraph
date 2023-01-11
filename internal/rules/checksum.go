package rules

import (
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"taskgraph/internal/hostfs"

	"github.com/bmatcuk/doublestar/v4"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Checksum struct {
	Inner        Rule
	WorkspaceDir string
	Stdout       io.Writer
}

// Execute implements Rule
func (c *Checksum) Execute(ctx context.Context) error {
	if len(c.Inner.Inputs()) == 0 {
		return c.Inner.Execute(ctx)
	}

	current, err := checksum(hostfs.FS(), c.Inner.Getwd(), c.Inputs(), []string{})
	if err != nil {
		return err
	}

	previous, err := c.load()
	if err != nil {
		return err
	}

	if current != previous {
		if err := c.Inner.Execute(ctx); err != nil {
			return err
		}
		if err := c.store(current); err != nil {
			logrus.Error(errors.Wrapf(err, "failed to store checksum for task %s", c.ID()))
		}
		return nil
	}

	fmt.Fprintf(c.Stdout, "%s is up-to-date\n", c.Inner.ID())
	return nil
}

// Dependencies implements Rule
func (c *Checksum) Dependencies() []string {
	return c.Inner.Dependencies()
}

// ID implements Rule
func (c *Checksum) ID() string {
	return c.Inner.ID()
}

// Inputs implements Rule
func (c *Checksum) Inputs() []string {
	return c.Inner.Inputs()
}

// Outputs implements Rule
func (c *Checksum) Outputs() []string {
	return c.Inner.Outputs()
}

func (c *Checksum) Getwd() string {
	return c.Inner.Getwd()
}

var _ Rule = &Checksum{}

func checksum(fs fs.FS, cwd string, includes []string, excludes []string) (string, error) {
	paths := mapset.NewSet[string]()

	for _, source := range includes {
		results, err := doublestar.Glob(fs, filepath.Join(cwd, source))
		if err != nil {
			return "", err
		}

		for _, path := range results {
			if f, _ := os.Stat(path); !f.IsDir() {
				relpath, _ := filepath.Rel(cwd, path)
				if !excluded(excludes, relpath) {
					paths.Add(path)
				}
			}
		}
	}

	p := paths.ToSlice()
	sort.Strings(p)

	h := crc32.NewIEEE()
	for _, path := range p {
		if _, err := io.WriteString(h, path); err != nil {
			return "", err
		}

		f, err := fs.Open(path)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func excluded(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if match, _ := doublestar.Match(filepath.ToSlash(pattern), filepath.ToSlash(path)); match {
			return true
		}
	}
	return false
}

func (t *Checksum) load() (string, error) {
	b, err := ioutil.ReadFile(t.path())
	if os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", errors.Wrap(err, "failed to load task checksum")
	}

	return strings.TrimSpace(string(b)), nil
}

func (t *Checksum) store(c string) error {
	path := t.path()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	if err := ioutil.WriteFile(path, []byte(c), 0644); err != nil {
		return errors.Wrap(err, "failed to write task checksum")

	}

	return nil
}

func (t *Checksum) path() string {
	return filepath.Join(t.WorkspaceDir, ".taskgraph", t.ID())
}
