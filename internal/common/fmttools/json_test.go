package fmttools_test

import (
	"testing"

	"code/internal/common/fmttools"

	"github.com/stretchr/testify/require"
)

func TestToJSON(t *testing.T) {
	type subTest struct {
		name         string
		data         any
		shouldIndent bool
		expected     string
		wantErr      bool
	}

	tests := []subTest{
		{
			name:     "compact JSON",
			data:     map[string]int{"count": 42},
			expected: `{"count":42}`,
		},
		{
			name:         "indented JSON",
			data:         map[string]int{"count": 42},
			shouldIndent: true,
			expected:     "{\n  \"count\": 42\n}",
		},
		{
			name:    "unsupported type",
			data:    make(chan int),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := fmttools.ToJSON(test.data, test.shouldIndent)

			if test.wantErr {
				require.Error(t, err)
				require.Nil(t, actual)

				return
			}

			require.NoError(t, err)
			require.JSONEq(t, test.expected, string(actual))
		})
	}
}
