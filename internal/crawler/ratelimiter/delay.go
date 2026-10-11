package ratelimiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

// DelayRateLimiter enforces a minimum delay between operations.
type DelayRateLimiter struct {
	delay    time.Duration
	nextTime time.Time
	sync.Mutex
}

// NewDelayRateLimiter creates a rate limiter with the specified delay.
func NewDelayRateLimiter(delay time.Duration) (*DelayRateLimiter, error) {
	if delay <= 0 {
		return nil, errors.New("delay must be positive")
	}

	return &DelayRateLimiter{
		delay: delay,
	}, nil
}

// Wait blocks for the configured delay or until ctx is canceled.
func (d *DelayRateLimiter) Wait(ctx context.Context) error {
	d.Lock()
	defer d.Unlock()

	now := time.Now()

	waitUntil := d.nextTime
	if waitUntil.Before(now) {
		waitUntil = now
	}

	timer := time.NewTimer(time.Until(waitUntil))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-timer.C:
		d.nextTime = time.Now().Add(d.delay)

		return nil
	}
}
