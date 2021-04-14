package proxy

import (
	"encoding/json"
)

type RequestAddr struct {
	Host      string
	Port      string
	Key       string
	Network   string
	Timestamp string
	Random    string
}

// MarshalBinary
func (r *RequestAddr) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

// UnmarshalBinary
func (r *RequestAddr) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &r)
}
