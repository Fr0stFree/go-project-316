// Package timeutils provides utility functions for working with time.
package timeutils

import "time"

// UTCNowPretty returns the current UTC time truncated to the nearest second.
func UTCNowPretty() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
