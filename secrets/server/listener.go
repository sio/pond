package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
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
	err = s.handleSSH(ctx, conn, chans)
	if err != nil {
		log.Printf("SSH session from %s failed: %v", tcp.RemoteAddr(), err)
	}
}

func (s *SecretServer) handleSSH(ctx context.Context, conn *ssh.ServerConn, chans <-chan ssh.NewChannel) error {
	select {
	case <-ctx.Done():
		return nil
	case incoming := <-chans:
		if incoming == nil {
			return nil
		}
		if incoming.ChannelType() != "session" {
			message := fmt.Sprintf("unknown channel type: %s", incoming.ChannelType())
			incoming.Reject(ssh.UnknownChannelType, msg)
			return fmt.Errorf(message)
		}
		ch, reqs, err := incoming.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept SSH channel: %w", err)
		}
		defer ch.Close()
		endpoint := getEndpoint(reqs)
		log.Printf("Detected API endpoint: %q", endpoint)
		go discardChannelRequests(ctx, reqs)
		query, err := io.ReadAll(ch)
		if err != nil {
			return fmt.Errorf("error while receiving API query: %w", err)
		}
		log.Printf("new API query (%d bytes): %q\n", len(query), string(query))
		ch.Write([]byte(strings.ToUpper(string(query))))
		return nil
	}
}

// Detect API endpoint based on requests currently queued in ssh channel
func getEndpoint(requests <-chan *ssh.Request) string {
	var endpoint string
loop:
	for {
		select {
		case r := <-requests:
			if r == nil {
				break loop
			}
			var allow bool
			switch r.Type {
			case "exec":
				endpoint = string(r.Payload[4:])
				allow = true
			case "shell", "pty-req":
				allow = true
			default:
				allow = false
			}
			r.Reply(allow, nil)
		case <-time.After(time.Second / 100): // we've exhaused pending requests queue
			break loop
		}
	}
	return endpoint
}

func discardChannelRequests(ctx context.Context, reqs <-chan *ssh.Request) {
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-reqs:
			if r == nil {
				return
			}
			r.Reply(false, nil)
		}
	}
}
