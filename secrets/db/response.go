package db

import (
	"encoding/json"
	"fmt"
)

type Response struct {
	Errors []string          `json:"errors"`
	Items  []json.RawMessage `json:"items,omitempty"`
}

func (r *Response) Add(item any) {
	raw, err := json.Marshal(item)
	if err != nil {
		r.Error("failed to encode %T to json (would be item #%d)", item, len(r.Items))
		return
	}
	r.Items = append(r.Items, json.RawMessage(raw))
}

// Send error message to end user.
//
// Golang error interface is intentionally not used here to avoid accidentally
// passing internal information to third parties
func (r *Response) Error(msg string, args ...any) {
	for index, arg := range args {
		if _, isErr := arg.(error); isErr {
			args[index] = "**redacted**"
		}
	}
	r.Errors = append(r.Errors, fmt.Sprintf(msg, args...))
}
