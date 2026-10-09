package ratelimiter

import (
	"context"
	"time"
)

type RPSRateLimiter struct {
	tokens chan struct{}
}

func NewRPSRateLimiter(ctx context.Context, rps int) (*RPSRateLimiter, error) {
	// validate rps
	l := RPSRateLimiter{
		tokens: make(chan struct{}, 1),
	}
	l.tokens <- struct{}{}

	interval := time.Second / time.Duration(rps)
	go l.refillTokens(ctx, interval)

	return &l, nil
}
func (r *RPSRateLimiter) refillTokens(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			select {
			case r.tokens <- struct{}{}:
				continue
			default:
			}
		}
	}
}
func (r *RPSRateLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.tokens:
		return nil
	}
}
