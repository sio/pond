package tests

import (
	"github.com/sio/pond/secrets/master"
	"github.com/sio/pond/secrets/util"
	"testing"

	"bytes"
	"crypto/rand"
	"encoding/base64"
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

		// ssh-keygen -Lf certPath | grep ID
		expectedBoxKey = "JgyCPNQAml3Lcm21zXfZPYIHiFw4I/1bjhxfbX5CyV0="
	)
	cert, err := util.LoadCertificate(certPath)
	if err != nil {
		t.Fatalf("LoadCertificate: %v", err)
	}
	if expectedBoxKey != cert.KeyId {
		t.Fatalf(
			"unexpected box public key:\nwant: %s\n got: %s",
			expectedBoxKey,
			cert.KeyId,
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
	boxPubKey, err := base64.StdEncoding.DecodeString(cert.KeyId)
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

// Measure encryption+decryption cycle.
// See BenchmarkMasterKeyEncrypt for a baseline.
func BenchmarkMasterKeyEncryptDecrypt(b *testing.B) {
	const (
		keyPath  = "keys/master"
		certPath = keyPath + ".cert"
	)
	cert, err := util.LoadCertificate(certPath)
	if err != nil {
		b.Fatalf("LoadCertificate: %v", err)
	}
	signer, err := LocalKey(keyPath)
	if err != nil {
		b.Fatalf("LocalKey: %v", err)
	}
	key, err := master.NewKey(signer, cert)
	if err != nil {
		b.Fatalf("NewKey: %v", err)
	}
	boxPubKey, err := base64.StdEncoding.DecodeString(cert.KeyId)
	if err != nil {
		b.Fatalf("base64 decode: %v", err)
	}
	var receiverPubKey = new([32]byte)
	copy(receiverPubKey[:], boxPubKey)
	senderPubKey, senderPrivKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatalf("GenerateKey: %v", err)
	}
	var nonce = new([24]byte)
	_, err = io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		b.Fatalf("rand: %v", err)
	}

	var (
		msgSize = 30
		buf     = make([]byte, 4096)
		pkg     = make([]byte, msgSize+box.Overhead+len(senderPubKey)+len(nonce))
	)
	copy(pkg, senderPubKey[:])
	copy(pkg[len(senderPubKey):], nonce[:])
	_, err = io.ReadFull(rand.Reader, buf)
	if err != nil {
		b.Fatalf("rand: %v", err)
	}
	for i := 0; i < b.N; i++ {
		start := i % (len(buf) - msgSize)
		send := buf[start : start+msgSize]
		pkg = box.Seal(pkg[:len(senderPubKey)+len(nonce)], send, nonce, receiverPubKey, senderPrivKey)
		msg, err := key.Decrypt(pkg)
		if err != nil {
			b.Fatalf("decrypt: %v", err)
		}
		if !bytes.Equal(msg, send) {
			b.Fatalf("mangled message:\nwant %x\n got: %x", send, msg)
		}
	}
}

// Measure encryption speed to provide baseline
// for BenchmarkMasterKeyEncryptDecrypt
func BenchmarkMasterKeyEncrypt(b *testing.B) {
	receiverPubKey, _, err := box.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatalf("GenerateKey: %v", err)
	}
	senderPubKey, senderPrivKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatalf("GenerateKey: %v", err)
	}
	var nonce = new([24]byte)
	_, err = io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		b.Fatalf("rand: %v", err)
	}

	var (
		msgSize = 30
		buf     = make([]byte, 4096)
		pkg     = make([]byte, msgSize+box.Overhead+len(senderPubKey)+len(nonce))
	)
	copy(pkg, senderPubKey[:])
	copy(pkg[len(senderPubKey):], nonce[:])
	_, err = io.ReadFull(rand.Reader, buf)
	if err != nil {
		b.Fatalf("rand: %v", err)
	}
	for i := 0; i < b.N; i++ {
		start := i % (len(buf) - msgSize)
		send := buf[start : start+msgSize]
		pkg = box.Seal(pkg[:len(senderPubKey)+len(nonce)], send, nonce, receiverPubKey, senderPrivKey)
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
