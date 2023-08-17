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
)

// Load public key from file system and return ssh-agent connection
// for the corresponding private key
func Open(path string) (*Conn, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(raw)
	if err != nil {
		return nil, err
	}
	return New(key)
}

// Return ssh-agent connection that corresponds to a given public key
func New(public ssh.PublicKey) (*Conn, error) {
	conn := &Conn{key: public}
	if err := conn.connect(); err != nil {
		return nil, err
	}
	return conn, nil
}

// SSH agent connection for a specific key
//
// Conn implements ssh.Signer interface and may be used everywhere
// ssh private key is expected
type Conn struct {
	key    ssh.PublicKey
	socket net.Conn
	agent  agent.Agent
	count  uint32 // TODO: expose as metrics
	mu     sync.Mutex
}

func (c *Conn) PublicKey() ssh.PublicKey {
	return c.key
}

func (c *Conn) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	sig, err := c.sign(rand, data)
	if err == nil {
		return sig, nil
	}
	err = c.connect()
	if err != nil {
		return nil, err
	}
	return c.sign(rand, data)
}

func (c *Conn) sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	if c.agent == nil || c.key == nil {
		return nil, fmt.Errorf("ssh-agent connection not initialized")
	}
	atomic.AddUint32(&c.count, 1)
	return c.agent.Sign(c.key, data)
}

func (c *Conn) Close() error {
	if c == nil || c.socket == nil {
		return nil
	}
	return c.socket.Close()
}

func (c *Conn) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Close()
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

	msg := make([]byte, 32)
	_, err = rand.Read(msg)
	if err != nil {
		return err
	}
	_, err = c.sign(rand.Reader, msg)
	if err == nil {
		return nil
	}

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
