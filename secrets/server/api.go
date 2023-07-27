package server

import (
	"context"
	"encoding/json"
	"errors"

	"secrets/db"
)

const defaultEndpoint = "secret"

// Prepare reply to API client
func (s *SecretServer) handleAPI(ctx context.Context, pubkey, endpoint string, body []byte) ([]byte, error) {
	var errs []error
	var e error
	var resp *db.Response

	resp, e = s.queryAPI(ctx, pubkey, endpoint, body)
	if resp == nil {
		resp = &db.Response{Errors: []string{"empty API response"}}
	}
	errs = append(errs, e)

	var raw []byte
	raw, e = json.Marshal(resp)
	errs = append(errs, e)

	if len(raw) > 0 && raw[len(raw)-1] != byte('\n') {
		raw = append(raw, byte('\n'))
	}

	return raw, errors.Join(errs...)
}

// API entrypoint
//
// Even though db.Response already contains Errors field, those messages are
// for the end user. We still need to pass lower level errors via second
// return value for development and administrative purposes.
func (s *SecretServer) queryAPI(ctx context.Context, pubkey, endpoint string, body []byte) (*db.Response, error) {
	response := &db.Response{}
	query := &db.Query{}
	err := json.Unmarshal(body, query)
	if err != nil {
		response.Error("failed to decode query JSON")
		return response, err
	}
	switch endpoint {
	case defaultEndpoint:
		return s.db.Execute(ctx, s.agent, pubkey, query)
	case "admin":
		return s.db.ExecuteAdmin(ctx, s.agent, pubkey, query)
	default:
		response.Error("invalid endpoint: %q", endpoint)
		return response, nil
	}
}
