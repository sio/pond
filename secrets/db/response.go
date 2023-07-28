package db

import (
	"encoding/json"
	"fmt"
)

// Note: replacing this with a generic Response[T any] while introduces type
// safety for Send() function is more trouble than it's worth. All receivers of
// *Response type would need to specify which instance of Response[T] they
// expect, and it's not that simple to declare beforehand.
type Response struct {
	Errors []string          `json:"errors"`
	Items  []json.RawMessage `json:"items,omitempty"`
}

// Send payload to end user
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

// Make sure that empty .Errors slice does not get serialized to JSON as "null"
// instead of "[]".
// Similar handling for .Items is not required because of "omitempty".
func (r *Response) MarshalJSON() ([]byte, error) {
	type responseData Response
	data := responseData(*r)
	if data.Errors == nil {
		data.Errors = make([]string, 0)
	}
	return json.Marshal(data)
}
