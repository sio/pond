package tests

import (
	"github.com/sio/pond/secrets/master"
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

const (
	keyPath  = "../../tests/keys/master"
	certPath = keyPath + ".cert"
)

func TestNewCertificate(t *testing.T) {
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
	_, err = file.Write(cert.Marshal())
	if err != nil {
		t.Fatalf("writing certificate to disk: %v", err)
	}
}

func TestMasterKey(t *testing.T) {
	const (
		// ssh-keygen -Lf certPath | grep ID
		expectedBoxKey = "JgyCPNQAml3Lcm21zXfZPYIHiFw4I/1bjhxfbX5CyV0="
	)
	cert, err := master.LoadCertificate(certPath)
	if err != nil {
		t.Fatalf("LoadCertificate: %v", err)
	}
	want, err := base64.StdEncoding.DecodeString(expectedBoxKey)
	if err != nil {
		t.Fatalf("base64: %v", err)
	}
	got := cert.SendTo()
	if !bytes.Equal(want, got[:]) {
		t.Fatalf(
			"unexpected box public key:\nwant: %x\n got: %x",
			want,
			got[:],
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

	const content = "hello world!"

	senderPubKey, senderPrivKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	receiverPubKey := cert.SendTo()
	var nonce = new([24]byte)
	_, err = io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		t.Fatalf("rand: %v", err)
	}
	message := box.Seal(nil, []byte(content), nonce, receiverPubKey, senderPrivKey)

	received, err := key.Unbox(message, senderPubKey, nonce)
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
	cert, err := master.LoadCertificate(certPath)
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
	receiverPubKey := cert.SendTo()
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
		pkg     = make([]byte, msgSize+box.Overhead)
	)
	_, err = io.ReadFull(rand.Reader, buf)
	if err != nil {
		b.Fatalf("rand: %v", err)
	}
	for i := 0; i < b.N; i++ {
		start := i % (len(buf) - msgSize)
		send := buf[start : start+msgSize]
		pkg = box.Seal(pkg[:0], send, nonce, receiverPubKey, senderPrivKey)
		msg, err := key.Unbox(pkg, senderPubKey, nonce)
		if err != nil {
			b.Logf("%x\n", pkg)
			b.Fatalf("iteration %d: decrypt: %v", i, err)
		}
		if !bytes.Equal(msg, send) {
			b.Fatalf("iteration %d: mangled message:\nwant %x\n got: %x", i, send, msg)
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
		pkg     = make([]byte, msgSize+box.Overhead)
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
		pkg = box.Seal(pkg[:], send, nonce, receiverPubKey, senderPrivKey)
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
