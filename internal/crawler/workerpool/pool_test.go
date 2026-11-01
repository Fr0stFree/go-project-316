package workerpool_test

import (
	"context"
	"slices"
	"testing"

	"code/internal/crawler/workerpool"

	"github.com/stretchr/testify/require"
)

func TestWorkerPool_Start(t *testing.T) {
	pool := workerpool.New[int, int](3)

	cancel := pool.Start(context.Background(), func(_ context.Context, job int) int {
		return job * 2
	}, 2)
	defer cancel()

	for _, job := range []int{1, 2, 3} {
		pool.Tasks() <- job
	}

	close(pool.Tasks())

	results := make([]int, 0, 3)
	for result := range pool.Results() {
		results = append(results, result)
	}

	slices.Sort(results)
	require.Equal(t, []int{2, 4, 6}, results)
}
