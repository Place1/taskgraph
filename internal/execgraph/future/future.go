package future

import (
	"sync"
)

type Future[T any] interface {
	Get() T
}

type future[T any] struct {
	wg    sync.WaitGroup
	value T
}

func (f *future[T]) Get() T {
	f.wg.Wait()

	return f.value
}

// Create a new Future running fn in a go routine.
func New[T any](fn func() T) Future[T] {

	f := future[T]{}
	f.wg.Add(1)

	go func() {
		f.value = fn()
		f.wg.Done()
	}()

	return &f
}

type valuefuture[T any] struct {
	value T
}

func Of[T any](value T) Future[T] {
	return valuefuture[T]{value}
}

func (f valuefuture[T]) Get() T {
	return f.value
}

// Turn a list of Futures into a single Future with a list of values.
func All[T any](xs []Future[T]) Future[[]T] {
	return New(
		func() []T {
			ret := make([]T, len(xs))

			for i := 0; i < len(xs); i++ {
				ret[i] = xs[i].Get()
			}

			return ret
		},
	)
}
