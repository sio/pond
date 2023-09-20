package value

import (
	"github.com/sio/pond/lib/bytepack"
	"github.com/sio/pond/secrets/master"

	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/nacl/box"
	"io"
)

const (
	// Maximum secret size is set to the upper bound on single message size
	// recommended by NaCl: https://pkg.go.dev/golang.org/x/crypto@v0.13.0/nacl/box#pkg-overview
	//
	// Support for larger secrets may be added in future if need arises
	MaxValueBytes = 16 * 1024

	paddingMaxBytes = 64
	nonceBytes      = 24
)

func (v *Value) Encrypt(master *master.Certificate, plaintext []byte) (err error) {
	if len(plaintext) > MaxValueBytes {
		return fmt.Errorf("secret values larger than %d bytes are not supported", MaxValueBytes)
	}
	var nonce = new([24]byte)
	_, err = io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		return err
	}

	senderPublic, senderPrivate, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	var padding = make([]byte, 1+paddingMaxBytes, 1+paddingMaxBytes+len(plaintext)+box.Overhead)
	_, err = io.ReadFull(rand.Reader, padding)
	if err != nil {
		return err
	}
	padding = padding[:1+int(padding[0])%paddingMaxBytes]

	cipher := box.Seal(nil, append(padding, plaintext...), nonce, master.SendTo(), senderPrivate)
	pack, err := bytepack.Pack([][]byte{
		senderPublic[:],
		nonce[:],
		cipher,
	})
	if err != nil {
		return err
	}
	v.blob = pack.Blob()
	return nil
}

func (v *Value) Decrypt(master *master.Key) (plaintext []byte, err error) {
	blob, err := bytepack.Wrap(v.blob)
	if err != nil {
		return nil, err
	}
	if blob.Size() != 3 {
		return nil, fmt.Errorf("unexpected number of elements unpacked from blob: %d instead of 3", blob.Size())
	}

	var sender = new([32]byte)
	copy(sender[:], blob.Element(0))

	var nonce = new([24]byte)
	copy(nonce[:], blob.Element(1))

	var cipher = blob.Element(2)
	plaintext, err = master.Unbox(cipher, sender, nonce)
	if err != nil {
		return nil, err
	}
	return plaintext[1+int(plaintext[0])%paddingMaxBytes:], nil
}
