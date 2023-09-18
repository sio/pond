package master

import (
	"errors"

	"golang.org/x/crypto/nacl/box"
)

// Open NaCl box addressed to master key
func (k *Key) Unbox(cipher []byte, sender *[32]byte, nonce *[24]byte) (plain []byte, err error) {
	_, secret, err := boxKey(k.signer, k.seed)
	defer clean(secret[:])
	if err != nil {
		return nil, err
	}
	plain, ok := box.Open(nil, cipher, nonce, sender, secret)
	if !ok {
		return nil, errors.New("message decryption failed")
	}
	return plain, nil
}
