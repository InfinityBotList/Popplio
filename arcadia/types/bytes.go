package types

import (
	"encoding/json"
	"fmt"
)

// Bytes is a Rust Vec<u8>.
//
// serde renders Vec<u8> as a JSON ARRAY OF NUMBERS, not as base64. Go's
// encoding/json does the opposite for []byte (base64 string), so a plain []byte
// here would silently mis-decode every CDN chunk upload.
type Bytes []byte

func (b *Bytes) UnmarshalJSON(data []byte) error {
	var nums []uint16
	if err := json.Unmarshal(data, &nums); err != nil {
		return fmt.Errorf("expected an array of byte values: %w", err)
	}

	out := make(Bytes, len(nums))
	for i, n := range nums {
		if n > 255 {
			return fmt.Errorf("invalid value: integer `%d`, expected u8", n)
		}

		out[i] = byte(n)
	}

	*b = out
	return nil
}

func (b Bytes) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte("[]"), nil
	}

	nums := make([]uint16, len(b))
	for i, v := range b {
		nums[i] = uint16(v)
	}

	return json.Marshal(nums)
}
