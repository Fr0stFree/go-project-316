package ratelimiter_test

import (
	"context"
	"testing"
	"time"

	"code/internal/crawler/ratelimiter"

	"github.com/stretchr/testify/require"
)

func TestNewRPSRateLimiter(t *testing.T) {
	type subTest struct {
		name    string
		rps     int
		wantErr bool
	}

	tests := []subTest{
		{
			name:    "negative RPS",
			rps:     -1,
			wantErr: true,
		},
		{
			name:    "zero RPS",
			rps:     0,
			wantErr: true,
		},
		{
			name: "minimum valid RPS",
			rps:  1,
		},
		{
			name: "valid RPS",
			rps:  100,
		},
		{
			name: "maximum valid RPS",
			rps:  1_000_000_000,
		},
		{
			name:    "RPS exceeds maximum",
			rps:     1_000_000_001,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			limiter, err := ratelimiter.NewRPSRateLimiter(ctx, test.rps)

			if test.wantErr {
				require.EqualError(t, err, "rps must be between 1 and 1000000000")
				require.Nil(t, limiter)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, limiter)
		})
	}
}

func prepareRPSRateLimiter(ctx context.Context, t *testing.T, rps int) *ratelimiter.RPSRateLimiter {
	t.Helper()

	limiter, err := ratelimiter.NewRPSRateLimiter(ctx, rps)
	require.NoError(t, err)

	return limiter
}

func TestRPSRateLimiter_Wait(t *testing.T) {
	t.Run("first request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		limiter := prepareRPSRateLimiter(ctx, t, 1)

		waitCtx, stop := context.WithTimeout(ctx, 100*time.Millisecond)
		defer stop()

		require.NoError(t, limiter.Wait(waitCtx))
	})

	t.Run("refill tokens", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		limiter := prepareRPSRateLimiter(ctx, t, 20)

		require.NoError(t, limiter.Wait(ctx))

		start := time.Now()

		require.NoError(t, limiter.Wait(ctx))

		elapsed := time.Since(start)
		require.GreaterOrEqual(t, elapsed, 40*time.Millisecond)
	})

	t.Run("context deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		limiter := prepareRPSRateLimiter(ctx, t, 1)

		require.NoError(t, limiter.Wait(ctx))

		waitCtx, stop := context.WithTimeout(ctx, 50*time.Millisecond)
		defer stop()

		err := limiter.Wait(waitCtx)

		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		limiter := prepareRPSRateLimiter(ctx, t, 1)

		require.NoError(t, limiter.Wait(ctx))

		waitCtx, stop := context.WithCancel(ctx)
		stop()

		err := limiter.Wait(waitCtx)

		require.ErrorIs(t, err, context.Canceled)
	})
}
