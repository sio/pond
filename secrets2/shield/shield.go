// Store sensitive value encrypted in-memory
//
// This provides a countermeasure to memory sidechannel attacks like Spectre,
// Meltdown, Rowhammer, etc. To retrieve shielded value the attacker would need
// to fetch a large (currently 128KB) prekey entirely without errors which is
// unlikely with current generation of attacks.
//
// Similar to how ssh-agent stores private key material:
//
//	https://xorhash.gitlab.io/xhblog/0010.html
package shield

import (
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/ssh"

	"secrets/util"
)

const (
	info                     = "pond/secrets: in-memory shield for sensitive data"
	prekeySize               = 128 * 1024
	prekeyMinReads    uint32 = 10
	prekeyMaxReads    uint32 = 100
	prekeyMinLifetime        = 3 * time.Second
	prekeyMaxLifetime        = 30 * time.Second
)

type Shield struct {
	signer        ssh.Signer
	lock          sync.RWMutex
	cipher        []byte
	prekey        []byte
	prekeyReads   atomic.Uint32
	prekeyExpires time.Time
	reshieldError error
}

func New(signer ssh.Signer, data []byte) (*Shield, error) {
	s := &Shield{
		signer: signer,
		prekey: make([]byte, prekeySize),
		cipher: make([]byte, len(data)+secretbox.Overhead),
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	e := s.shield(data)
	if e != nil {
		return nil, e
	}
	return s, nil
}

// Read sensitive value protected by shield.
//
// Do not copy this data and don't forget to call ShieldedValue.Close()
// when you're done.
func (s *Shield) Value() (*ShieldedValue, error) {
	if s.reshieldError != nil {
		s.reshieldError = s.reshield() // retry reshielding before surfacing the error
		if s.reshieldError != nil {
			return nil, s.reshieldError
		}
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	value, err := s.unshield()
	if err != nil {
		cleanup(value)
		return nil, err
	}

	if s.prekeyReads.Add(1) > prekeyMaxReads || time.Now().After(s.prekeyExpires) {
		go func() { s.reshieldError = s.reshield() }()
	}

	sv := ShieldedValue(value)
	return &sv, nil
}

// Reencrypt shielded data with a new random prekey
func (s *Shield) reshield() error {
	if s.prekeyReads.Load() < prekeyMinReads &&
		s.prekeyExpires.Sub(time.Now()) > prekeyMaxLifetime-prekeyMinLifetime {
		// Due to locking reshield() may have been queued multiple times for one expiration,
		// exit early from all the duplicated calls.
		return nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	data, err := s.unshield()
	defer cleanup(data)
	if err != nil {
		return err
	}
	err = s.shield(data)
	if err != nil {
		return err
	}
	return nil
}

// Derive encryption parameters from prekey
func (s *Shield) derive(key *[32]byte, nonce *[24]byte) (err error) {
	hash := sha512.New()
	_, err = hash.Write(s.prekey)
	if err != nil {
		return err
	}
	sig, err := s.signer.Sign(util.AntiReader, s.prekey)
	if err == util.ErrAntiReader {
		return fmt.Errorf("signature not deterministic: %s", s.signer.PublicKey().Type())
	}
	if err != nil {
		return err
	}
	_, err = hash.Write(sig.Blob)
	if err != nil {
		return err
	}
	h := hash.Sum(nil)
	copy((*key)[:], h[:])
	copy((*nonce)[:], h[len(*key):])
	return nil
}

// Encrypt sensitive data in memory
func (s *Shield) shield(d []byte) (err error) {
	if len(d)+secretbox.Overhead != len(s.cipher) {
		return fmt.Errorf("shield not initialized [%d bytes instead of %d]", len(s.cipher), len(d)+secretbox.Overhead)
	}

	_, err = io.ReadFull(rand.Reader, s.prekey)
	if err != nil {
		return err
	}

	var key [32]byte
	var nonce [24]byte
	defer cleanup(key[:])
	s.derive(&key, &nonce)

	ptr := &s.cipher[0]
	s.cipher = secretbox.Seal(s.cipher[:0], d, &nonce, &key)
	if ptr != &s.cipher[0] {
		return errors.New("storage array reallocated unexpectedly")
	}
	cleanup(d)

	s.prekeyReads.Store(0)
	s.prekeyExpires = time.Now().Add(prekeyMaxLifetime)
	return nil
}

// Append sensitive data to provided slice
//
// Do not forget to clear the data after use to minimize the possibility of leaks.
// See cleanup() function in this module
func (s *Shield) unshield() (data []byte, err error) {
	if len(s.cipher) < secretbox.Overhead {
		return nil, fmt.Errorf("shield not initialized [%d bytes]", len(s.cipher))
	}
	data = make([]byte, len(s.cipher)-secretbox.Overhead)
	ptr := &data[0]
	var key [32]byte
	var nonce [24]byte
	defer cleanup(key[:])
	s.derive(&key, &nonce)
	data, ok := secretbox.Open(data[:0], s.cipher, &nonce, &key)
	if ptr != &data[0] {
		return nil, errors.New("data array reallocated while unshielding")
	}
	if !ok {
		return nil, errors.New("decryption failed")
	}
	return data, nil
}

// Clear sensitive data from memory even before the garbage collector kicks in
func cleanup(sensitive []byte) {
	for i := 0; i < len(sensitive); i++ {
		sensitive[i] = 0
	}
	_, _ = rand.Read(sensitive)
}
