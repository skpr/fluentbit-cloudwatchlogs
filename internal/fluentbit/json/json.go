package json

import (
	"encoding/json"
	"io"
)

// Parse "json" payloads seny by Fluent Bit.
func Parse(r io.Reader) ([]Line, error) {
	var l []Line

	err := json.NewDecoder(r).Decode(&l)
	if err != nil {
		return l, err
	}

	return l, nil
}
