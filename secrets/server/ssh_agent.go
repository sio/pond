package server

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"sync/atomic"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func newSSHAgentConn(publicKeyPath string) (*sshAgentConn, error) {
	raw, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(raw)
	if err != nil {
		return nil, err
	}
	conn := &sshAgentConn{key: key}
	if err := conn.connect(); err != nil {
		return nil, err
	}
	return conn, nil
}

type sshAgentConn struct {
	key    ssh.PublicKey
	socket net.Conn
	agent  agent.Agent
	count  uint32 // TODO: expose as metrics
}

func (c *sshAgentConn) PublicKey() ssh.PublicKey {
	return c.key
}

func (c *sshAgentConn) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
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

func (c *sshAgentConn) sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	if c.agent == nil || c.key == nil {
		return nil, fmt.Errorf("ssh-agent connection not initialized")
	}
	atomic.AddUint32(&c.count, 1)
	return c.agent.Sign(c.key, data)
}

func (c *sshAgentConn) Close() error {
	if c == nil || c.socket == nil {
		return nil
	}
	return c.socket.Close()
}

func (c *sshAgentConn) connect() error {
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
	if err != nil {
		return fmt.Errorf("ssh-agent initialization failed: %w", err)
	}
	return nil
}
