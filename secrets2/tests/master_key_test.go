package tests

import (
	"secrets/master"
	"testing"

	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
)

func TestNewCertificate(t *testing.T) {
	const (
		keyPath  = "keys/master"
		certPath = keyPath + ".cert"
	)
	signer, err := LocalKey(keyPath)
	if err != nil {
		t.Fatalf("LocalKey: %v", err)
	}
	cert, err := master.NewCertificate(signer)
	if err != nil {
		t.Fatalf("NewCertificate: %v", err)
	}
	_, err = os.Stat(certPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if err == nil {
		return
	}
	file, err := os.OpenFile(certPath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	_, err = file.Write(ssh.MarshalAuthorizedKey(cert))
	if err != nil {
		t.Fatalf("writing certificate to disk: %v", err)
	}
}

func TestMasterKey(t *testing.T) {
	const (
		keyPath  = "keys/master"
		certPath = keyPath + ".cert"

		// ssh-keygen -Lf certPath | grep sendto
		expectedBoxKey = "0000002c44305248446a474f734137685550614c75454d53642b79416b48595a6265774172482f4330562f6c6131383d"
	)
	b64PubKey, err := hex.DecodeString(expectedBoxKey)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	cert, err := LocalCert(certPath)
	if err != nil {
		t.Fatalf("LocalCert: %v", err)
	}
	if string(b64PubKey[4:]) != cert.Permissions.Extensions["sendto:master@pond/secrets"] {
		t.Fatalf(
			"unexpected box public key:\nwant: %s\n got: %s",
			string(b64PubKey[4:]),
			cert.Permissions.Extensions["sendto:master@pond/secrets"],
		)
	}
	signer, err := LocalKey(keyPath)
	if err != nil {
		t.Fatalf("LocalKey: %v", err)
	}
	key, err := master.NewKey(signer, cert)
	if err != nil {
		t.Fatalf("NewKey: %v", err)
	}
	boxPubKey, err := base64.StdEncoding.DecodeString(
		cert.Permissions.Extensions["sendto:master@pond/secrets"],
	)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}

	const content = "hello world!"

	senderPubKey, senderPrivKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	var receiverPubKey = new([32]byte)
	copy(receiverPubKey[:], boxPubKey)
	var nonce = new([24]byte)
	_, err = io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		t.Fatalf("rand: %v", err)
	}
	message := append([]byte{}, senderPubKey[:]...)
	message = append(message, nonce[:]...)
	message = box.Seal(message, []byte(content), nonce, receiverPubKey, senderPrivKey)

	received, err := key.Decrypt(message)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(received) != content {
		t.Fatalf("received mangled message: %q", string(received))
	}
}

// Use local private key for signing messages in tests
func LocalKey(keyname string) (ssh.Signer, error) {
	raw, err := os.ReadFile(keyname)
	if err != nil {
		return nil, err
	}
	key, err := ssh.ParseRawPrivateKey(raw)
	if err != nil {
		return nil, err
	}
	private, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return nil, fmt.Errorf("%T can not be used for signatures: %w", key, err)
	}
	return private, nil
}

// Load certificate from file system
func LocalCert(path string) (*ssh.Certificate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(raw)
	if err != nil {
		return nil, err
	}
	cert, ok := pubkey.(*ssh.Certificate)
	if !ok {
		return nil, fmt.Errorf("not a certificate: %s", pubkey.Type())
	}
	return cert, nil
}
