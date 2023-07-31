package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"secrets/db"
)

const (
	// The endpoint to use when one not specified explicitly
	defaultEndpoint = "secret"

	// Hard size limit for incoming queries
	//
	// Current value (20KiB) is rather conservative and may be increased in the
	// future. It was chosen based on 16KiB recommendation for maximum
	// nacl.SecretBox message size plus some overhead for API query scaffolding.
	//
	// Links:
	//	https://pkg.go.dev/golang.org/x/crypto@v0.11.0/nacl/secretbox
	//	https://go-review.googlesource.com/c/crypto/+/35910
	maxQueryBytes = 20 * 1024
)

var (
	apiOK = errors.New("OK")
)

// Prepare reply to API client
func (s *SecretServer) handleAPI(ctx context.Context, pubkey, endpoint string, body io.Reader) ([]byte, error) {
	var errs []error
	var e error
	var resp *db.Response

	resp, e = s.queryAPI(ctx, pubkey, endpoint, body)
	if resp == nil {
		resp = &db.Response{Errors: []string{"empty response"}}
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
func (s *SecretServer) queryAPI(ctx context.Context, pubkey, endpoint string, body io.Reader) (*db.Response, error) {
	response := &db.Response{}

	var err error
	var buf bytes.Buffer
	_, err = io.CopyN(&buf, body, maxQueryBytes)
	if err == nil {
		response.Errorf("request must be shorter than %d bytes", maxQueryBytes)
		return response, fmt.Errorf("request is too large")
	}
	if !errors.Is(err, io.EOF) {
		response.Errorf("query sending failed")
		return response, fmt.Errorf("reading request body: %w", err)
	}
	if buf.Len() == 0 {
		response.Errorf("received empty query")
		return response, response.LastError()
	}
	query := &db.Query{}
	err = json.Unmarshal(buf.Bytes(), query)
	if err != nil {
		response.Errorf("failed to decode query JSON")
		return response, fmt.Errorf("decoding query JSON: %w", err)
	}
	switch endpoint {
	case defaultEndpoint:
		response, err = s.db.Execute(ctx, pubkey, query)
	case "admin":
		response, err = s.db.ExecuteAdmin(ctx, pubkey, query)
	default:
		response.Errorf("invalid endpoint: %q", endpoint)
		err = response.LastError()
	}
	if err == nil {
		err = apiOK
	}
	return response, fmt.Errorf("%s/%s/%s: %w", endpoint, query.Action, query.Namespace, err)
}
