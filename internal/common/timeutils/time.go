// Package timeutils provides utility functions for working with time.
package timeutils

import "time"

// UTCNowPretty returns the provided clock's UTC time truncated to the nearest second.
func UTCNowPretty(now func() time.Time) time.Time {
	return now().UTC().Truncate(time.Second)
}
