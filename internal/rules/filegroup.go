package rules

import "context"

type Filegroup struct {
	IID  string
	Srcs []string
	Cwd  string
}

// Execute implements Rule
func (*Filegroup) Execute(ctx context.Context) error {
	return nil
}

// Dependencies implements Rule
func (*Filegroup) Dependencies() []string {
	return []string{}
}

// Inputs implements Rule
func (f *Filegroup) Inputs() []string {
	return f.Srcs
}

// Outputs implements Rule
func (f *Filegroup) Outputs() []string {
	return f.Srcs
}

func (t *Filegroup) ID() string {
	return t.IID
}

func (t *Filegroup) Getwd() string {
	return t.Cwd
}

var _ Rule = &Filegroup{}
