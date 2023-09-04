// Manupulate encrypted secret values
package value

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"
)

// Encrypted secret value
type Value struct {
	Path      []string
	Blob      []byte
	Created   time.Time
	Expires   time.Time
	Signer    ssh.PublicKey
	Signature []byte
}

const (
	fileHeader    = "[pond/secrets]"
	sigHeader     = "pond/secrets: Encrypted secret value"
	sigNonceBytes = 64
)

// Signature formats (when not same as public key format)
var sigFormat = map[string]string{
	ssh.KeyAlgoRSA: ssh.KeyAlgoRSASHA512,
}

func (v *Value) bytesToSign(nonce []byte) []byte {
	var buf = new(bytes.Buffer)
	if len(nonce) == sigNonceBytes {
		_, _ = buf.Write(nonce)
	} else {
		_, err := io.CopyN(buf, rand.Reader, sigNonceBytes)
		if err != nil {
			panic("crypto/rand: " + err.Error())
		}
	}
	_, _ = fmt.Fprintln(buf, sigHeader)
	_, _ = fmt.Fprintln(buf, v.Created.Unix())
	_, _ = fmt.Fprintln(buf, v.Expires.Unix())
	for _, p := range v.Path {
		_, _ = fmt.Fprintln(buf, p)
	}
	_, _ = buf.Write(v.Blob)
	return buf.Bytes()
}

// Add signature to value
func (v *Value) Sign(s ssh.Signer) error {
	data := v.bytesToSign(nil)
	sig, err := s.Sign(rand.Reader, data)
	if err != nil {
		return err
	}

	pubkey := s.PublicKey()
	expectedFormat, ok := sigFormat[pubkey.Type()]
	if !ok {
		expectedFormat = pubkey.Type()
	}
	if sig.Format != expectedFormat {
		return fmt.Errorf("received unsupported signature format: %s (instead of %s)", sig.Format, expectedFormat)
	}

	v.Signer = pubkey
	v.Signature = make([]byte, sigNonceBytes+len(sig.Blob))
	copy(v.Signature[:sigNonceBytes], data)
	copy(v.Signature[sigNonceBytes:], sig.Blob)
	return nil
}

// Verify value signature
func (v *Value) Verify() error {
	if v.Signer == nil || len(v.Signature) == 0 {
		return errors.New("value not signed yet")
	}
	data := v.bytesToSign(v.Signature[:sigNonceBytes])
	format, ok := sigFormat[v.Signer.Type()]
	if !ok {
		format = v.Signer.Type()
	}
	sig := &ssh.Signature{
		Format: format,
		Blob:   v.Signature[sigNonceBytes:],
	}
	return v.Signer.Verify(data, sig)
}
