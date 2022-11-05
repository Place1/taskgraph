package rules

import "context"

type Rule interface {
	ID() string
	Inputs() []string
	Outputs() []string
	Dependencies() []string
	Getwd() string
	Execute(ctx context.Context) error
}
