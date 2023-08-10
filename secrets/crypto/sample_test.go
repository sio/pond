package crypto

import (
	"testing"

	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
)

// Check that we can decrypt messages encrypted with Python implementation
func TestReadingFromPython(t *testing.T) {
	const (
		samplePath  = "sample_py_v1.json"
		decryptWith = "../tests/keys/storage"
	)
	sampleRaw, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatal(err)
	}
	var collection SampleCollection
	if err = json.Unmarshal(sampleRaw, &collection); err != nil {
		t.Fatal(err)
	}
	signer, err := LocalKey(decryptWith)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := signer.Sign(rand.Reader, []byte(samplePath))
	if err != nil {
		t.Fatal(err)
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(collection.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	if err = pubKey.Verify([]byte(samplePath), challenge); err != nil {
		t.Fatalf("failed to verify test signature: %v", err)
	}
	for index, sample := range collection.Samples {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			signature, err := v1signature(signer, sample.Keywords, sample.SignatureNonce)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(signature, sample.Signature) {
				t.Fatalf("could not reproduce signature from alt implementation")
			} else {
				t.Log("OK: signatures match")
			}
			key, err := v1kdf(signature, sample.KdfNonce)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(key[:], sample.Key) {
				t.Fatalf("could not reproduce HKDF result from alt implementation")
			} else {
				t.Log("OK: HKDF keys match")
			}
			decrypted, err := Decrypt(signer, sample.Keywords, sample.Encrypted)
			if err != nil {
				t.Fatal(err)
			}
			if string(decrypted) != sample.Message {
				t.Fatalf("mismatched decryption result: got %q, want %q", decrypted, sample.Message)
			}
		})
	}
}

type SampleCollection struct {
	PublicKey string   `json:"key"`
	Samples   []Sample `json:"samples"`
}

type Sample struct {
	Message        string
	Keywords       []string
	SignatureNonce []byte
	Signature      []byte
	KdfNonce       []byte
	Key            []byte
	Encrypted      []byte
}

type sampleBase64 struct {
	Message              string   `json:"message"`
	Keywords             []string `json:"keywords"`
	SignatureNonceBase64 string   `json:"signature_nonce"`
	SignatureBase64      string   `json:"signature"`
	KdfNonceBase64       string   `json:"kdf_nonce"`
	KeyBase64            string   `json:"key"`
	EncryptedBase64      string   `json:"encrypted"`
}

func (s *Sample) UnmarshalJSON(data []byte) error {
	var err error
	var temp sampleBase64
	if err = json.Unmarshal(data, &temp); err != nil {
		return err
	}
	s.Message = temp.Message
	s.Keywords = temp.Keywords

	s.SignatureNonce, err = base64.StdEncoding.DecodeString(temp.SignatureNonceBase64)
	if err != nil {
		return fmt.Errorf("failed to decode SignatureNonce from base64: %w", err)
	}

	s.Signature, err = base64.StdEncoding.DecodeString(temp.SignatureBase64)
	if err != nil {
		return fmt.Errorf("failed to decode Signature from base64: %w", err)
	}

	s.KdfNonce, err = base64.StdEncoding.DecodeString(temp.KdfNonceBase64)
	if err != nil {
		return fmt.Errorf("failed to decode KdfNonce from base64: %w", err)
	}

	s.Key, err = base64.StdEncoding.DecodeString(temp.KeyBase64)
	if err != nil {
		return fmt.Errorf("failed to decode Key from base64: %w", err)
	}

	s.Encrypted, err = base64.StdEncoding.DecodeString(temp.EncryptedBase64)
	if err != nil {
		return fmt.Errorf("failed to decode Encrypted from base64: %w", err)
	}
	return nil
}
