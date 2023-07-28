package db

import (
	"encoding/json"
	"fmt"
)

type Response struct {
	Errors []string          `json:"errors"`
	Items  []json.RawMessage `json:"items,omitempty"`
}

func (r *Response) Send(item any) {
	raw, err := json.Marshal(item)
	if err != nil {
		r.Errorf("failed to encode %T to json (would be item #%d)", item, len(r.Items))
		return
	}
	r.Items = append(r.Items, json.RawMessage(raw))
}

// Send error message to end user.
//
// Golang error interface is intentionally not used here to avoid accidentally
// passing internal information to third parties
func (r *Response) Errorf(msg string, args ...any) {
	for index, arg := range args {
		if _, isErr := arg.(error); isErr {
			args[index] = "**redacted**"
		}
	}
	r.Errors = append(r.Errors, fmt.Sprintf(msg, args...))
}

func (r *Response) LastError() error {
	if len(r.Errors) == 0 {
		return nil
	}
	return fmt.Errorf(r.Errors[len(r.Errors)-1])
}
