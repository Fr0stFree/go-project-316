// Package ratelimiter provides rate limiting utilities for controlling the frequency of HTTP requests.
package ratelimiter

import (
	"context"
	"errors"
	"time"
)

// RPSRateLimiter limits the rate of operations using a token bucket.
type RPSRateLimiter struct {
	tokens chan struct{}
}

// NewRPSRateLimiter creates a rate limiter that allows up to rps operations per second.
func NewRPSRateLimiter(ctx context.Context, rps int) (*RPSRateLimiter, error) {
	if rps <= 0 || time.Duration(rps) > time.Second {
		return nil, errors.New("rps must be more than 1")
	}

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

// Wait blocks until a token becomes available or ctx is canceled.
func (r *RPSRateLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.tokens:
		return nil
	}
}
