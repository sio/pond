// Interact with ssh-agent
package agent

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/sio/pond/secrets/util"
)

// Load public key from file system and return ssh-agent connection
// for the corresponding private key
func Open(path string) (*Conn, error) {
	key, err := util.LoadPublicKey(path)
	if err != nil {
		return nil, err
	}
	return New(key)
}

// Return ssh-agent connection that corresponds to a given public key
func New(key ssh.PublicKey) (*Conn, error) {
	var err error
	var conn = new(Conn)
	if key == nil {
		err = conn.connect()
	} else {
		err = conn.SetIdentity(key)
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// SSH agent connection for a specific key
//
// Conn implements ssh.Signer interface and may be used everywhere
// ssh private key is expected
type Conn struct {
	key        ssh.PublicKey
	socket     net.Conn
	agent      agent.ExtendedAgent
	flags      agent.SignatureFlags
	count      uint32 // TODO: expose as metrics
	lock       sync.RWMutex
	randomized bool
}

var _ ssh.Signer = new(Conn)

// Return public key for currently selected identity
func (c *Conn) PublicKey() ssh.PublicKey {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.key
}

// List available identities
func (c *Conn) ListKeys() []ssh.PublicKey {
	if c.agent == nil {
		panic("agent not initialized")
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	signers, err := c.agent.Signers()
	if err != nil {
		return nil
	}
	keys := make([]ssh.PublicKey, len(signers))
	for i := 0; i < len(signers); i++ {
		keys[i] = signers[i].PublicKey()
	}
	return keys
}

// Change identity that will be used for signing
func (c *Conn) SetIdentity(key ssh.PublicKey) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if key == nil {
		c.key = nil
		c.flags = 0
		return nil
	}

	oldKey := c.key
	oldFlags := c.flags

	c.key = key
	switch c.key.Type() {
	case ssh.KeyAlgoRSA, ssh.CertAlgoRSAv01: // do not use old SHA-1 signatures
		c.flags = agent.SignatureFlagRsaSha512
	}
	err := c.check(c.connect())
	if err != nil {
		c.key = oldKey
		c.flags = oldFlags
		return err
	}
	return nil
}

func (c *Conn) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	if c.randomized {
		// Let caller know that signature is not deterministic by consuming
		// a single byte from rand
		_, err := io.CopyN(io.Discard, rand, 1)
		if err != nil {
			return nil, err
		}
	}

	c.lock.RLock()
	sig, err := c.sign(data)
	c.lock.RUnlock()
	if err == nil {
		return sig, nil
	}

	c.lock.Lock()
	err = c.check(c.connect())
	c.lock.Unlock()
	if err != nil {
		return nil, err
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.sign(data)
}

func (c *Conn) sign(data []byte) (*ssh.Signature, error) {
	if c.agent == nil || c.key == nil {
		return nil, fmt.Errorf("ssh-agent connection not initialized")
	}
	atomic.AddUint32(&c.count, 1)
	return c.agent.SignWithFlags(c.key, data, c.flags)
}

func (c *Conn) Close() error {
	if c == nil || c.socket == nil {
		return nil
	}
	return c.socket.Close()
}

// Establish connection to ssh-agent
//
// Should be called mostly like this:
//
//	c.check(c.connect())
func (c *Conn) connect() error {
	err := c.Close()
	if err != nil {
		return err
	}
	saddr := os.Getenv("SSH_AUTH_SOCK")
	if saddr == "" {
		return fmt.Errorf("environment variable not set: SSH_AUTH_SOCK")
	}
	socket, err := net.Dial("unix", saddr)
	if err != nil {
		return err
	}
	c.socket = socket
	c.agent = agent.NewClient(socket)
	return nil
}

// Check that ssh-agent connection is usable
//
// Should be called mostly like this:
//
//	c.check(c.connect())
func (c *Conn) check(err error) error {
	if err != nil {
		return err
	}
	msg := make([]byte, 32)
	_, err = rand.Read(msg)
	if err != nil {
		return err
	}
	sig1, err := c.sign(msg)
	if err != nil {
		// Debug why signature failed to provide a better error message
		signers, dbgErr := c.agent.Signers()
		if dbgErr != nil {
			return err
		}
		for _, s := range signers {
			if bytes.Equal(s.PublicKey().Marshal(), c.key.Marshal()) {
				// ssh-agent contains required identity, error was about something else
				return err
			}
		}
		return fmt.Errorf("ssh-agent: identity not available: %s", ssh.FingerprintSHA256(c.key))
	}

	// Detect non-deterministic signatures
	sig2, err := c.sign(msg)
	if err != nil {
		return err
	}
	if !bytes.Equal(sig1.Blob, sig2.Blob) {
		c.randomized = true
	}
	return nil
}
