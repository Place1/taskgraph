package output

import (
	"hash/fnv"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/kr/text"
)

type std struct {
}

func NewStd() OutputFactory {
	return &std{}
}

var colors = []*color.Color{
	color.New(color.FgBlue),
	color.New(color.FgMagenta),
	color.New(color.FgCyan),
	color.New(color.FgWhite),
	color.New(color.FgYellow),
	color.New(color.FgHiBlue),
	color.New(color.FgHiMagenta),
	color.New(color.FgHiCyan),
	color.New(color.FgHiWhite),
	color.New(color.FgHiYellow),
}

// Stderr implements Factory
func (*std) Stderr(prefix string) io.Writer {
	c := colors[hash(prefix)%len(colors)]
	return text.NewIndentWriter(os.Stderr, []byte(c.Sprintf("[%s] ", prefix)))
}

// Stdout implements Factory
func (*std) Stdout(prefix string) io.Writer {
	c := colors[hash(prefix)%len(colors)]
	return text.NewIndentWriter(os.Stdout, []byte(c.Sprintf("[%s] ", prefix)))
}

func hash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}

var _ OutputFactory = &std{}
