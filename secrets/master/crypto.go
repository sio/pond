package master

import (
	"errors"
	"time"

	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/access"
)

const (
	boxKeySize   = 32
	boxNonceSize = 24
)

// Decrypt short messages addressed to master key
func (k *Key) Decrypt(message []byte) (content []byte, err error) {
	const minSize = boxKeySize + boxNonceSize + box.Overhead
	if len(message) <= minSize {
		return nil, errors.New("message is too short to decrypt")
	}

	var senderKey = new([boxKeySize]byte)
	copy(senderKey[:], message[:boxKeySize])

	var boxNonce = new([boxNonceSize]byte)
	copy(boxNonce[:], message[boxKeySize:boxKeySize+boxNonceSize])

	_, receiverKey, err := boxKey(k.signer, k.seed)
	defer clean(receiverKey[:])
	if err != nil {
		return nil, err
	}

	content, ok := box.Open(nil, message[boxKeySize+boxNonceSize:], boxNonce, senderKey, receiverKey)
	if !ok {
		return nil, errors.New("message decryption failed")
	}
	return content, nil
}

// Delegate administrative capabilities
func (k *Key) Delegate(
	to ssh.PublicKey,
	caps []access.Capability,
	paths []string,
	name string,
	lifetime time.Duration,
) (*access.Certificate, error) {
	return access.DelegateAdmin(k.signer, to, caps, paths, name, lifetime)
}
