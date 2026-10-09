package ratelimiter

import (
	"context"
	"sync"
	"time"
)

type DelayRateLimiter struct {
	delay time.Duration
	sync.Mutex
}

func NewDelayRateLimiter(delay time.Duration) (*DelayRateLimiter, error) {
	// todo: validate delay
	return &DelayRateLimiter{
		delay: delay,
	}, nil
}

func (d *DelayRateLimiter) Wait(ctx context.Context) error {
	d.Lock()
	defer d.Unlock()

	timer := time.NewTimer(d.delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
