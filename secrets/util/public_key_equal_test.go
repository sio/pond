package util

import (
	"testing"

	"crypto/ed25519"
	"crypto/rand"
	"golang.org/x/crypto/ssh"
	"io"
)

func TestPubkeyEqual(t *testing.T) {
	const plain = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGdbcW5tXesw+Aghy5PQPHZEHloqGS3wH6zyZDl45aFq"
	const withComment = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGdbcW5tXesw+Aghy5PQPHZEHloqGS3wH6zyZDl45aFq INSECURE TEST KEY"
	keyPlain, _, _, _, err := ssh.ParseAuthorizedKey([]byte(plain))
	if err != nil {
		t.Fatal(err)
	}
	keyComment, _, _, _, err := ssh.ParseAuthorizedKey([]byte(withComment))
	if err != nil {
		t.Fatal(err)
	}
	if !EqualSSH(keyPlain, keyComment) {
		t.Fatal("adding comment produced a key not equal to original")
	}
}

func BenchmarkPubkeyEqual(b *testing.B) {
	seedA := randomSeed()
	keys := [...]ssh.PublicKey{
		key(seedA), // compare to self
		key(seedA), // compare to equal key
		key(nil),   // compare to different key
	}
	for i := 0; i < b.N; i++ {
		index := i % len(keys)
		want := index < 2
		got := EqualSSH(keys[0], keys[index])
		if want != got {
			b.Errorf("invalid output from EqualSSH: want %v, got %v", want, got)
		}
	}
}

func randomSeed() []byte {
	seed := make([]byte, ed25519.SeedSize)
	_, err := io.ReadFull(rand.Reader, seed)
	if err != nil {
		panic(err)
	}
	return seed
}

func key(seed []byte) ssh.PublicKey {
	if seed == nil {
		seed = randomSeed()
	}
	cryptokey := ed25519.NewKeyFromSeed(seed)
	sshkey, err := ssh.NewSignerFromSigner(cryptokey)
	if err != nil {
		panic(err)
	}
	return sshkey.PublicKey()
}
