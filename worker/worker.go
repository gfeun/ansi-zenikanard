package worker

import "sync"

// Pool represents a set of worker doing the same task
type Pool struct {
	sync.WaitGroup
	fn        func()
	instances int
}

// New returns a worker
func New(fn func(), instances int) Pool {
	return Pool{fn: fn, instances: instances}
}

// Run starts "w.instances" number of goroutine running w.fn function
func (w *Pool) Run() {
	for i := 0; i < w.instances; i++ {
		w.Add(1)
		go func() {
			defer w.Done()
			w.fn()
		}()
	}
}
