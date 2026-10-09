// Package workerpool provides a generic worker pool for concurrent task processing.
package workerpool

import (
	"context"
	"sync"
)

// WorkerPool processes jobs concurrently using a fixed number of workers.
type WorkerPool[T, R any] struct {
	jobs    chan T
	results chan R
	process func(context.Context, T) R
	size    int
}

// New creates a WorkerPool with the specified number of workers
// and a function for processing each job.
func New[T, R any](
	size int,
	process func(context.Context, T) R,
) *WorkerPool[T, R] {
	return &WorkerPool[T, R]{
		jobs:    make(chan T, 100), // TODO: hide?
		results: make(chan R, 100),
		size:    size,
		process: process,
	}
}

// Jobs returns a send-only channel for submitting jobs to the pool.
func (p *WorkerPool[T, R]) Jobs() chan<- T {
	return p.jobs
}

// Results returns a receive-only channel for collecting processed results.
func (p *WorkerPool[T, R]) Results() <-chan R {
	return p.results
}

// Start launches the configured number of workers.
func (p *WorkerPool[T, R]) Start(ctx context.Context) context.CancelFunc {
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)

	for range p.size {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-p.jobs:
					if !ok {
						return
					}

					select {
					case p.results <- p.process(ctx, job):
						continue
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(p.results)
	}()

	return cancel
}
