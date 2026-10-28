// Package timeutils provides utility functions for working with time.
package timeutils

import "time"

// UTCTruncatedToSeconds returns the provided clock's UTC time truncated to the nearest second.
func UTCTruncatedToSeconds(clock func() time.Time) time.Time {
	return clock().UTC().Truncate(time.Second)
}
