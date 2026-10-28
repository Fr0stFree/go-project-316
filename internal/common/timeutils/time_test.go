package timeutils_test

import (
	"testing"
	"time"

	"code/internal/common/timeutils"

	"github.com/stretchr/testify/require"
)

func TestUTCTruncatedToSeconds(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "truncate nanoseconds",
			input:    time.Date(2026, 10, 28, 12, 30, 45, 123456789, time.UTC),
			expected: time.Date(2026, 10, 28, 12, 30, 45, 0, time.UTC),
		},
		{
			name: "convert to UTC",
			input: time.Date(
				2026, 10, 28, 15, 30, 45, 0,
				time.FixedZone("UTC+3", 3*60*60),
			),
			expected: time.Date(2026, 10, 28, 12, 30, 45, 0, time.UTC),
		},
		{
			name:     "already normalized",
			input:    time.Date(2026, 10, 28, 12, 30, 45, 0, time.UTC),
			expected: time.Date(2026, 10, 28, 12, 30, 45, 0, time.UTC),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := timeutils.UTCTruncatedToSeconds(func() time.Time {
				return test.input
			})

			require.Equal(t, test.expected, actual)
			require.Equal(t, time.UTC, actual.Location())
		})
	}
}
