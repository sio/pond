package master

import (
	"errors"

	"golang.org/x/crypto/nacl/box"
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
