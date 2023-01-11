package output

import "io"

type OutputFactory interface {
	Stdout(prefix string) io.Writer
	Stderr(prefix string) io.Writer
}
