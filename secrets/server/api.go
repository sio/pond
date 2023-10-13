package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

func (s *Server) handleAPI(ctx context.Context, key ssh.PublicKey, request io.Reader) *response {
	var resp = newResponse()
	var query []name
	err := json.NewDecoder(request).Decode(&query)
	if err != nil {
		if errors.Is(err, io.EOF) {
			err = errors.New("empty query")
		}
		resp.Errorf("invalid json: %v", err)
		return resp
	}
	if len(query) == 0 {
		resp.Errorf("empty query")
		return resp
	}
	allowed := s.acl.AllowedRead(key)
	for _, key := range query {
		value, err := s.repo.Search(string(key), allowed)
		if err != nil {
			resp.Errorf("%s: %v", key, err)
			continue
		}
		plaintext, err := value.Decrypt(s.master)
		if err != nil {
			resp.Errorf("%s: %v", key, err)
			continue
		}
		resp.Set(key, secret(plaintext))
	}
	return resp
}

type secret string

type name string

func newResponse() *response {
	return &response{
		Secrets: make(map[name]secret),
	}
}

type response struct {
	Secrets map[name]secret `json:"secrets"`
	Errors  multiError      `json:"errors"`
}

func (r *response) Set(n name, s secret) {
	r.Secrets[n] = s
}

func (r *response) Errorf(f string, args ...any) {
	r.Errors.Errorf(f, args...)
}

func (r *response) Send(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

func (r *response) Status() string {
	if r == nil {
		return "NIL"
	}
	if len(r.Errors) == 0 {
		return "OK"
	}
	tag := "client error"
	if len(r.Secrets) != 0 {
		tag = "client error (partial)"
	}
	return fmt.Sprintf("%s: %s", tag, r.Errors)
}
