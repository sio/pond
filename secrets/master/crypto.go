package master

import (
	"errors"

	"golang.org/x/crypto/nacl/box"
)

// Open NaCl box addressed to master key
func (k *Key) Unbox(cipher []byte, sender *[32]byte, nonce *[24]byte) (plain []byte, err error) {
	// Box key is stored in memory only for as long as it is in use, each call
	// retriggers box key derivation and each new key is stored at new
	// location.
	//
	// This provides a slight countermeasure for side channel attacks which
	// leak partial memory regions: an attacker can never be sure when and
	// where the key will be stored, reducing the time key is stored also
	// reduces the chance that such attack would be successful.
	//
	// Storing plaintext key in RAM is still vulnerable to attacks based on
	// memory dumps.
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
