// Package fmttools provides utility functions for formatting data
package fmttools

import (
	"encoding/json"
)

// ToJSON converts the given data to a JSON byte slice. If the indent parameter is true, the JSON output will be indented for better readability.
func ToJSON(data any, shouldIndent bool) ([]byte, error) {
	if shouldIndent {
		return json.MarshalIndent(data, "", "  ")
	}

	return json.Marshal(data)
}
