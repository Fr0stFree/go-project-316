// Package fmttools provides utility functions for formatting data
package fmttools

import (
	"encoding/json"
)

// ToJSON converts the given data to a JSON byte slice. If the indent parameter is true, the JSON output will be indented for better readability.
func ToJSON(data any, shouldIndent bool) ([]byte, error) {
	var (
		jsonData []byte
		err      error
	)

	if shouldIndent {
		jsonData, err = json.MarshalIndent(data, "", "  ")
	} else {
		jsonData, err = json.Marshal(data)
	}

	if err != nil {
		return nil, err
	}

	return jsonData, nil
}
