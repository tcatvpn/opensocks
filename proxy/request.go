package proxy

import (
	"encoding/json"
)

// The Request address struct
type RequestAddr struct {
	Host      string
	Port      string
	Key       string
	Network   string
	Timestamp string
	Random    string
}

// MarshalBinary marshals the RequestAddr
func (r *RequestAddr) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

// UmarshalBinary unmarshals the RequestAddr
func (r *RequestAddr) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &r)
}
