//go:build test_cli

package cli

import (
	"github.com/sio/pond/lib/sandbox"

	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Start ssh-agent server and load private keys
func sshAgent(box *sandbox.Sandbox, key ...string) (*agentServer, error) {
	const innerSocket = "/ssh-agent.sock"
	box.Setenv("SSH_AUTH_SOCK", innerSocket)
	socket, err := box.Path(innerSocket)
	if err != nil {
		return nil, err
	}
	server := &agentServer{
		socket: socket,
	}
	for _, k := range key {
		err = server.LoadKey(k)
		if err != nil {
			return nil, err
		}
	}
	go server.Serve()
	return server, server.Err()
}

// ssh-agent server that is accessible both inside and outside the sandbox
type agentServer struct {
	socket   string
	keys     map[string]ssh.Signer
	listener *net.UnixListener
	cancel   context.CancelFunc
	err      error
}

var _ agent.Agent = new(agentServer)

func (s *agentServer) Serve() {
	if s.listener != nil {
		panic("agentServer is not reentrant")
	}
	l, err := net.Listen("unix", s.socket)
	if err != nil {
		s.err = err
		return
	}
	defer func() { _ = l.Close() }()
	s.listener = l.(*net.UnixListener)

	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())

	for {
		select {
		case <-ctx.Done():
			s.err = ctx.Err()
			return
		default:
		}
		err := s.listener.SetDeadline(time.Now().Add(time.Second))
		if err != nil {
			s.err = err
			return
		}
		conn, err := s.listener.Accept()
		if os.IsTimeout(err) {
			continue
		}
		if err != nil {
			s.err = err
			return
		}
		go func() {
			_ = agent.ServeAgent(s, conn)
			_ = conn.Close()
		}()
	}
}

func (s *agentServer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
}

func (s *agentServer) LoadKey(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	key, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return err
	}
	fingerprint := ssh.FingerprintSHA256(key.PublicKey())
	if s.keys == nil {
		s.keys = make(map[string]ssh.Signer)
	}
	s.keys[fingerprint] = key
	return nil
}

func (s *agentServer) Err() error {
	return s.err
}

func (s *agentServer) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	fingerprint := ssh.FingerprintSHA256(key)
	private, exists := s.keys[fingerprint]
	if !exists {
		return nil, fmt.Errorf("ssh-agent: key not found: %s", fingerprint)
	}
	return private.Sign(rand.Reader, data)
}

func (s *agentServer) List() ([]*agent.Key, error) {
	keys := make([]*agent.Key, len(s.keys))
	var i int
	for _, k := range s.keys {
		keys[i] = sshKeyToAgentKey(k.PublicKey())
		i++
	}
	return keys, nil
}

func sshKeyToAgentKey(pub ssh.PublicKey) *agent.Key {
	return &agent.Key{
		Format:  pub.Type(),
		Blob:    pub.Marshal(),
		Comment: "",
	}
}

// The rest of ssh-agent features are intentionally not implemented.
// We have no need for them in our tests.
func (s *agentServer) Add(key agent.AddedKey) error {
	panic("not implemented: Add()")
}
func (s *agentServer) Remove(key ssh.PublicKey) error {
	panic("not implemented: Remove()")
}
func (s *agentServer) RemoveAll() error {
	panic("not implemented: RemoveAll()")
}
func (s *agentServer) Lock(passphrase []byte) error {
	panic("not implemented: Lock()")
}
func (s *agentServer) Unlock(passphrase []byte) error {
	panic("not implemented: Unlock()")
}
func (s *agentServer) Signers() ([]ssh.Signer, error) {
	panic("not implemented: Signers()")
}
