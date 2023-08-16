package master

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/ssh"

	"secrets/access"
	"secrets/shield"
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

	var senderKey [boxKeySize]byte
	n := copy(senderKey[:], message[:boxKeySize])
	if n != boxKeySize {
		return nil, errors.New("failed to read sender's public key")
	}

	var boxNonce [boxNonceSize]byte
	n = copy(boxNonce[:], message[boxKeySize:boxKeySize+boxNonceSize])
	if n != boxNonceSize {
		return nil, errors.New("failed to read message nonce")
	}

	boxkey, err := k.boxkey.Value()
	if err != nil {
		return nil, fmt.Errorf("unshield: %w", err)
	}
	var receiverKey [boxKeySize]byte
	copy(receiverKey[:], boxkey.Bytes())
	boxkey.Close()
	defer shield.Clean(receiverKey[:])

	content, ok := box.Open(nil, message[boxKeySize+boxNonceSize:], &boxNonce, &senderKey, &receiverKey)
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
) (*ssh.Certificate, error) {
	return access.DelegateAdmin(k.signer, to, caps, paths, name, lifetime)
}
