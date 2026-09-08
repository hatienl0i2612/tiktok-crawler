// Package jsonvalue contains JSON scalar types shared by TikTok response
// models. TikTok inconsistently encodes some numeric fields as either JSON
// numbers or decimal strings.
package jsonvalue

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// Int64 accepts an integer encoded as a JSON number or decimal string. JSON
// null is treated as zero.
type Int64 int64

// UnmarshalJSON implements json.Unmarshaler.
func (value *Int64) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*value = 0
		return nil
	}
	if data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		parsed, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return err
		}
		*value = Int64(parsed)
		return nil
	}
	parsed, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	*value = Int64(parsed)
	return nil
}
