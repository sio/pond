package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	connectionTimeout = 3 * time.Second
)

// Initialize SecretServer
func New(publicKeyPath string) (*SecretServer, error) {
	agent, err := newSSHAgentConn(publicKeyPath)
	if err != nil {
		return nil, err
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, pubkey ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{
				Extensions: map[string]string{
					"pubkey": string(pubkey.Marshal()),
				},
			}, nil
		},
	}
	config.AddHostKey(agent)
	return &SecretServer{
		agent:  agent,
		config: config,
	}, nil
}

type SecretServer struct {
	config *ssh.ServerConfig
	agent  *sshAgentConn
}

func (s *SecretServer) Run(ctx context.Context, address string) error {
	defer s.agent.Close()
	l, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}
	defer l.Close()
	listener, ok := l.(*net.TCPListener)
	if !ok {
		return fmt.Errorf("not a TCPListener: %T", l)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := listener.SetDeadline(time.Now().Add(connectionTimeout)); err != nil {
			return err
		}
		conn, err := listener.Accept()
		if os.IsTimeout(err) {
			continue
		}
		if err != nil {
			log.Printf("failed to accept TCP connection: %v", err)
			continue
		}
		go s.handleTCP(ctx, conn)
	}
	return nil
}

func (s *SecretServer) handleTCP(ctx context.Context, tcp net.Conn) {
	defer tcp.Close()
	if err := tcp.SetDeadline(time.Now().Add(connectionTimeout)); err != nil {
		log.Printf("failed to set TCP deadline: %v", err)
		return
	}
	conn, chans, reqs, err := ssh.NewServerConn(tcp, s.config)
	if err != nil {
		log.Printf("failed to accept SSH connection from %s: %v", tcp.RemoteAddr(), err)
		return
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)
	s.handleSSH(ctx, conn, chans)
}

func (s *SecretServer) handleSSH(ctx context.Context, conn *ssh.ServerConn, chans <-chan ssh.NewChannel) {
	log.Println("TODO: handle SSH connections")
}
