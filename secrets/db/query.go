package db

import (
	"encoding/json"
	"fmt"
)

type Query struct {
	Action    queryAction      `json:"action"`
	Namespace string           `json:"namespace"`
	Items     *json.RawMessage `json:"items"`
}

func (q *Query) String() string {
	if q == nil {
		return "{nil Query}"
	}
	return fmt.Sprintf(
		`{Query action=%q namespace=%q items=%s}`,
		q.Action,
		q.Namespace,
		string(*q.Items),
	)
}

type queryAction string

const (
	Get    queryAction = "get"
	Set    queryAction = "set"
	List   queryAction = "list"
	Delete queryAction = "delete"
)

func (qa *queryAction) UnmarshalText(text []byte) error {
	value := queryAction(text)
	switch value {
	case Get, Set, List, Delete:
		*qa = value
		return nil
	default:
		return fmt.Errorf("invalid query action: %q", string(text))
	}
}

func (qa queryAction) MarshalText() ([]byte, error) {
	return []byte(qa), nil
}
