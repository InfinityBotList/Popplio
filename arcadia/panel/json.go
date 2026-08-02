package panel

import (
	"bytes"
	"encoding/json"
)

// jsonString encodes a string without Go's default HTML escaping, so the panel
// receives the same bytes serde_json would have produced.
func jsonString(s string) ([]byte, error) {
	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(s); err != nil {
		return nil, err
	}

	// Encode appends a newline.
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
