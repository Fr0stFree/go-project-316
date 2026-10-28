package ratelimiter

import (
	"context"
	"errors"
	"time"
)

// DelayRateLimiter enforces a minimum delay between operations.
type DelayRateLimiter struct {
	delay    time.Duration
	nextTime time.Time
	token    chan struct{}
}

// NewDelayRateLimiter creates a rate limiter with the specified delay.
func NewDelayRateLimiter(delay time.Duration) (*DelayRateLimiter, error) {
	if delay <= 0 {
		return nil, errors.New("delay must be positive")
	}

	limiter := &DelayRateLimiter{
		delay: delay,
		token: make(chan struct{}, 1),
	}
	limiter.restoreToken()

	return limiter, nil
}

// Wait blocks until the next operation is permitted or ctx is canceled.
func (d *DelayRateLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-d.token:
	}

	defer d.restoreToken()

	now := time.Now()

	waitUntil := d.nextTime
	if waitUntil.After(now) {
		timer := time.NewTimer(time.Until(waitUntil))
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	d.nextTime = time.Now().Add(d.delay)

	return nil
}

func (d *DelayRateLimiter) restoreToken() {
	d.token <- struct{}{}
}
