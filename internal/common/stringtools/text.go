// Package stringtools provides utility functions for working with strings.
package stringtools

import "strings"

// CleanText normalizes whitespace in the input text by replacing whitespaces with a single space.
func CleanText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
