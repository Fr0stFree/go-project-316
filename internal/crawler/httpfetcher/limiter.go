package httpfetcher

import "context"

type rateLimiter interface {
	Wait(ctx context.Context) error
}

type noopLimiter struct{}

func (n *noopLimiter) Wait(_ context.Context) error {
	return nil
}
