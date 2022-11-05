// https://github.com/golang/go/issues/47803
package hostfs

import (
	"io/fs"
	"os"
)

var osFS = new(hostFS)

func FS() fs.FS {
	return osFS
}

type hostFS struct{}

func (*hostFS) Open(name string) (fs.File, error)          { return os.Open(name) }
func (*hostFS) ReadDir(name string) ([]fs.DirEntry, error) { return os.ReadDir(name) }
func (*hostFS) Stat(name string) (fs.FileInfo, error)      { return os.Stat(name) }
