package certs

import (
	"golang.org/x/crypto/ssh"

	"crypto"
	"fmt"
	"os"
	"reflect"
	"unsafe"
)

func PublicKey(filename string) (crypto.PublicKey, error) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	sshkey, _, _, _, err := ssh.ParseAuthorizedKey(raw)
	if err != nil {
		return nil, err
	}
	crypto, ok := sshkey.(ssh.CryptoPublicKey)
	if !ok {
		return nil, fmt.Errorf("unable to convert ssh key to crypto.PublicKey: %T", sshkey)
	}
	return crypto.CryptoPublicKey(), nil
}

func PrivateKey(filename string) (crypto.Signer, error) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, err
	}

	// Obtain value of unexported field: wrappedSigner{signer, pubKey}, panic if anything goes wrong
	// https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.13.0:ssh/keys.go;l=947
	elem := reflect.ValueOf(signer).Elem()
	field := elem.FieldByName("signer")
	value := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
	return value.(crypto.Signer), nil
}
