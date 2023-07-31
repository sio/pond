package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"

	"secrets/db"
	"secrets/util"
)

const (
	connectionTimeout = 3 * time.Second
)

// Initialize SecretServer
func New(publicKeyPath, databasePath string) (*SecretServer, error) {
	agent, err := newSSHAgentConn(publicKeyPath)
	if err != nil {
		return nil, err
	}
	db, err := db.Open(databasePath, agent)
	if err != nil {
		return nil, err
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, pubkey ssh.PublicKey) (*ssh.Permissions, error) {
			keyText := util.KeyText(pubkey)
			if err := db.AllowAPI(keyText); err != nil {
				return nil, err
			}
			return &ssh.Permissions{
				Extensions: map[string]string{
					"pubkey": keyText,
				},
			}, nil
		},
	}
	config.AddHostKey(agent)
	return &SecretServer{
		config: config,
		db:     db,
	}, nil
}

type SecretServer struct {
	config *ssh.ServerConfig
	db     *db.Database
}

func (s *SecretServer) Run(ctx context.Context, address string) error {
	defer s.db.Close()
	l, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}
	defer l.Close()
	listener, ok := l.(*net.TCPListener)
	if !ok {
		return fmt.Errorf("not a TCPListener: %T", l)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for s := range signals {
			_, _ = fmt.Fprintln(os.Stderr, "")
			log.Printf("Caught %s, initiating graceful exit...", s)
			cancel()
		}
	}()

	log.Printf("Listening on %s", listener.Addr())
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
			log.Printf("TCP accept: %v", err)
			continue
		}
		go s.handleTCP(ctx, conn)
	}
	return nil
}

func (s *SecretServer) handleTCP(ctx context.Context, tcp net.Conn) {
	defer tcp.Close()
	if err := tcp.SetDeadline(time.Now().Add(connectionTimeout)); err != nil {
		log.Printf("TCP deadline: %v", err)
		return
	}
	conn, chans, reqs, err := ssh.NewServerConn(tcp, s.config)
	if err != nil {
		log.Printf("Deny SSH from %s: %v", tcp.RemoteAddr(), err)
		return
	}
	defer conn.Close()
	go discardRequests(ctx, reqs)
	err = s.handleSSH(ctx, conn, chans)
	if err != nil {
		log.Printf("Client %s: %v", tcp.RemoteAddr(), err)
	} else {
		log.Printf("Client %s: OK", tcp.RemoteAddr())
	}
}

func (s *SecretServer) handleSSH(ctx context.Context, conn *ssh.ServerConn, chans <-chan ssh.NewChannel) error {
	select {
	case <-ctx.Done():
		return nil
	case incoming, ok := <-chans:
		if !ok || incoming == nil {
			return nil
		}
		if incoming.ChannelType() != "session" {
			message := fmt.Sprintf("unknown channel type: %s", incoming.ChannelType())
			incoming.Reject(ssh.UnknownChannelType, message)
			return fmt.Errorf(message)
		}
		ch, reqs, err := incoming.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept SSH channel: %w", err)
		}
		defer ch.Close()

		pubkey := conn.Permissions.Extensions["pubkey"]
		endpoint := getEndpoint(reqs)
		go discardRequests(ctx, reqs) // must go right after getEndpoint()

		var errs util.MultiError
		resp, err := s.handleAPI(ctx, pubkey, endpoint, ch)
		errs.Error(err)

		_, err = ch.Write(resp)
		errs.Errorf("writing to SSH channel: %w", err)

		err = ch.CloseWrite()
		errs.Errorf("closing SSH channel for writing: %w", err)

		_, err = ch.SendRequest("eow@openssh.com", false, nil)
		errs.Errorf("sending eow: %w", err)

		_, err = ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
		errs.Errorf("sending exit-status: %w", err)

		return errs.Sum()
	}
}

// Detect API endpoint based on requests currently queued in ssh channel
func getEndpoint(requests <-chan *ssh.Request) string {
	var endpoint = defaultEndpoint
loop:
	for {
		select {
		case r, ok := <-requests:
			if !ok {
				break loop
			}
			if r == nil {
				continue
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
			if r.WantReply {
				r.Reply(allow, nil)
			}
		case <-time.After(time.Second / 100): // we've exhaused pending requests queue
			break loop
		}
	}
	return endpoint
}

// Similar to ssh.DiscardRequests but less prone to goroutine leaks,
// this function exits on context cancellation and on closing requests channel.
func discardRequests(ctx context.Context, reqs <-chan *ssh.Request) {
	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-reqs:
			if !ok {
				return
			}
			if r == nil {
				continue
			}
			if r.WantReply {
				r.Reply(false, nil)
			}
		}
	}
}
