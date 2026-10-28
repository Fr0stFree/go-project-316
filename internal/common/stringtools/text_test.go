package stringtools_test

import (
	"testing"

	"code/internal/common/stringtools"

	"github.com/stretchr/testify/require"
)

func TestCleanText(t *testing.T) {
	type subTest struct {
		name     string
		input    string
		expected string
	}

	tests := []subTest{
		{
			name:     "normal text",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "multiple spaces",
			input:    "Hello    world",
			expected: "Hello world",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "  Hello world  ",
			expected: "Hello world",
		},
		{
			name:     "mixed whitespace",
			input:    "Hello\t\nworld\r\nGo",
			expected: "Hello world Go",
		},
		{
			name:     "empty and whitespace-only text",
			input:    " \t\n ",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := stringtools.CleanText(test.input)

			require.Equal(t, test.expected, actual)
		})
	}
}
