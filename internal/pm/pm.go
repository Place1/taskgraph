package pm

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"
)

type ProcessManager interface {
	Start(f func(ctx context.Context) error)
	Wait() error
}

type pm struct {
	wg    *errgroup.Group
	wgctx context.Context
}

func New(ctx context.Context) ProcessManager {
	wg, wgctx := errgroup.WithContext(ctx)
	return &pm{
		wg:    wg,
		wgctx: wgctx,
	}
}

// Start implements ProcessManager
func (p *pm) Start(f func(ctx context.Context) error) {
	p.wg.Go(func() error {
		err := f(p.wgctx)
		return err
	})
}

// Wait implements ProcessManager
func (p *pm) Wait() error {
	if err := p.wg.Wait(); errors.Is(err, context.Canceled) {
		return nil
	} else {
		return err
	}
}

var _ ProcessManager = &pm{}
