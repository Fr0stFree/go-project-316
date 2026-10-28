package ratelimiter_test

import (
	"context"
	"testing"
	"time"

	"code/internal/crawler/ratelimiter"

	"github.com/stretchr/testify/require"
)

func TestNewDelayRateLimiter(t *testing.T) {
	type subTest struct {
		name    string
		delay   time.Duration
		wantErr bool
	}

	tests := []subTest{
		{
			name:    "negative delay",
			delay:   -time.Second,
			wantErr: true,
		},
		{
			name:    "zero delay",
			delay:   0,
			wantErr: true,
		},
		{
			name:  "valid delay",
			delay: time.Millisecond,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limiter, err := ratelimiter.NewDelayRateLimiter(test.delay)

			if test.wantErr {
				require.Error(t, err)
				require.Nil(t, limiter)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, limiter)
		})
	}
}

func prepareRateLimiter(t *testing.T, delay time.Duration) *ratelimiter.DelayRateLimiter {
	t.Helper()

	limiter, err := ratelimiter.NewDelayRateLimiter(delay)
	require.NoError(t, err)

	return limiter
}

func TestDelayRateLimiter_Wait(t *testing.T) {
	t.Run("first request", func(t *testing.T) {
		limiter := prepareRateLimiter(t, time.Second)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		require.NoError(t, limiter.Wait(ctx))
	})

	t.Run("enforce delay", func(t *testing.T) {
		delay := 100 * time.Millisecond
		limiter := prepareRateLimiter(t, delay)

		require.NoError(t, limiter.Wait(context.Background()))

		start := time.Now()

		require.NoError(t, limiter.Wait(context.Background()))

		elapsed := time.Since(start)
		require.GreaterOrEqual(t, elapsed, delay)
	})

	t.Run("cancel during delay", func(t *testing.T) {
		limiter := prepareRateLimiter(t, time.Second)

		require.NoError(t, limiter.Wait(context.Background()))

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := limiter.Wait(ctx)

		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
