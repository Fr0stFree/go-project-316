package workerpool

import (
	"context"
	"sync"
)

// WorkerPool concurrently processes crawl jobs.
type WorkerPool[T, R any] struct {
	wg      sync.WaitGroup
	jobs    chan T
	results chan R
	process func(context.Context, T) R
	size    int
}

// NewWorkerPool creates a new worker pool.
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

func (p *WorkerPool[T, R]) Jobs() chan<- T {
	return p.jobs
}

func (p *WorkerPool[T, R]) Results() <-chan R {
	return p.results
}

// Start starts the specified number of workers.
func (p *WorkerPool[T, R]) Start(ctx context.Context) {
	for range p.size {
		p.wg.Add(1)
		go p.runWorker(ctx)
	}
}

// Stop stops the worker pool and waits for all workers to finish.
func (p *WorkerPool[T, R]) Stop() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

func (p *WorkerPool[T, R]) runWorker(ctx context.Context) {
	defer p.wg.Done()

	for job := range p.jobs {
		result := p.process(ctx, job)

		select {
		case p.results <- result:
		case <-ctx.Done():
			return
		}
	}
}
