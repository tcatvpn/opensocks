package proxy

import (
	"encoding/json"
)

type RequestAddr struct {
	Host      string
	Port      string
	Username  string
	Password  string
	Network   string
	Timestamp string
}

// MarshalBinary
func (e *RequestAddr) MarshalBinary() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalBinary
func (e *RequestAddr) UnmarshalBinary(data []byte) error {
	if err := json.Unmarshal(data, &e); err != nil {
		return err
	}
	return nil
}
