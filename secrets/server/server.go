package server

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/access"
	"github.com/sio/pond/secrets/master"
	"github.com/sio/pond/secrets/repo"
)

const (
	// Time before the server will break the Accept() call
	// to check if context has expired
	acceptTimeout = time.Second * 1

	// Time before the server will drop client connection
	connTimeout = time.Second * 5

	// Give client some time to gracefully close ssh connection
	// before forcefully severing it server-side
	closeTimeout = time.Second * 1
)

func Run(listen, repository string) error {
	srv, err := New(listen, repository)
	if err != nil {
		return err
	}
	return srv.Run(context.Background())
}

// Initialize secretd server listening at provided address
// to serve secrets from provided repository path
func New(listen, repository string) (*Server, error) {
	addr, err := url.Parse(listen)
	if err != nil {
		return nil, err
	}
	s := &Server{
		proto: addr.Scheme,
	}
	switch addr.Scheme {
	case "unix":
		s.addr = addr.Path
	case "tcp", "tcp4", "tcp6":
		s.addr = addr.Host
	case "ssh":
		s.proto = "tcp"
		s.addr = addr.Host
	default:
		return nil, fmt.Errorf("listening on %s not implemented", addr.Scheme)
	}
	s.repo, err = repo.Open(repository)
	if err != nil {
		return nil, err
	}
	s.acl, err = access.Open(s.repo.MasterCert())
	if err != nil {
		return nil, err
	}
	warnings, err := s.acl.Load(s.repo.AdminCerts(), s.repo.UserCerts())
	if err != nil {
		return nil, err
	}
	for _, w := range warnings {
		s.log(w)
	}
	s.master, err = master.Open(s.repo.MasterCert())
	if err != nil {
		return nil, err
	}
	s.ssh = &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			err := s.acl.Check(key, access.Read, "/")
			if err != nil {
				return nil, err
			}
			return &ssh.Permissions{
				Extensions: map[string]string{
					"key": string(key.Marshal()),
				},
			}, nil
		},
	}
	cert, err := ephemeralHostCert(s.master)
	if err != nil {
		return nil, err
	}
	s.ssh.AddHostKey(cert) // TODO: renew certificate in background when it's close to expiration
	return s, nil
}

type Server struct {
	proto, addr string
	ssh         *ssh.ServerConfig
	acl         *access.ACL
	repo        *repo.Repository
	master      *master.Key
	stop        bool
}

func (s *Server) Run(ctx context.Context) error {
	defer func() { _ = s.acl.Close() }()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range signals {
			s.log("Received %s, initiating graceful exit...", sig)
			s.stop = true
			cancel()
		}
	}()

	l, err := net.Listen(s.proto, s.addr) // TODO: ListenConfig.Listen supports passing context
	if err != nil {
		return fmt.Errorf("listening %s: %w", s.proto, err)
	}
	listener, ok := l.(deadlineListener)
	if !ok {
		return fmt.Errorf("listener does not support deadline: %T", l)
	}
	s.log("Listening on %s", listener.Addr())

	for {
		select {
		case <-ctx.Done():
			if s.stop {
				return nil
			}
			return ctx.Err()
		default:
		}
		err = listener.SetDeadline(time.Now().Add(acceptTimeout))
		if err != nil {
			return err
		}
		conn, err := listener.Accept()
		if os.IsTimeout(err) {
			continue
		}
		if err != nil {
			s.log(err)
			continue
		}
		go s.handleTCP(ctx, conn)
	}
}

func (s *Server) handleTCP(ctx context.Context, tcp net.Conn) {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	defer func() { _ = tcp.Close() }()
	err := tcp.SetDeadline(time.Now().Add(connTimeout))
	if err != nil {
		return
	}

	conn, chans, reqs, err := ssh.NewServerConn(tcp, s.ssh)
	if err != nil {
		s.log("%s: deny connection: %v", tcp.RemoteAddr(), err)
		return
	}
	defer func() { _ = conn.Close() }()

	go discard(ctx, reqs)
	status, err := s.handleSSH(ctx, conn, chans)
	if err != nil {
		s.log("%s: server error: %v", tcp.RemoteAddr(), err)
	} else {
		s.log("%s: %s", tcp.RemoteAddr(), status)
	}

	// Allow client to end connection gracefully
	go func() {
		_ = conn.Wait()
		cancel()
	}()
	select {
	case <-time.After(closeTimeout):
		s.log("%s: client did not end ssh session gracefully, terminating", tcp.RemoteAddr())
	case <-ctx.Done():
	}
}

func (s *Server) handleSSH(ctx context.Context, conn *ssh.ServerConn, chans <-chan ssh.NewChannel) (status string, err error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case incoming, ok := <-chans:
		if !ok || incoming == nil {
			return "", nil
		}
		go reject(ctx, chans) // we only care about the first channel
		if incoming.ChannelType() != "session" {
			message := fmt.Sprintf("unknown channel type: %s", incoming.ChannelType())
			_ = incoming.Reject(ssh.UnknownChannelType, message)
			return "", fmt.Errorf(message)
		}
		ch, reqs, err := incoming.Accept()
		if err != nil {
			return "", fmt.Errorf("failed to accept SSH channel: %w", err)
		}
		defer func() { _ = ch.Close() }()
		go discard(ctx, reqs)

		key, err := ssh.ParsePublicKey([]byte(conn.Permissions.Extensions["key"]))
		if err != nil {
			return "", fmt.Errorf("parsing client public key: %w", err)
		}

		var errs multiError
		resp := s.handleAPI(ctx, key, ch)
		err = resp.Send(ch)
		errs.Errorf("sending SSH response: %w", err)

		err = ch.CloseWrite()
		errs.Errorf("closing SSH channel for writing: %w", err)

		_, err = ch.SendRequest("eow@openssh.com", false, nil)
		errs.Errorf("sending eow: %w", err)

		var zero [4]byte
		_, err = ch.SendRequest("exit-status", false, zero[:])
		errs.Errorf("sending exit-status: %w", err)

		return resp.Status(), errs.Sum()
	}
}

func (s *Server) log(f any, args ...any) {
	format, ok := f.(string)
	if !ok {
		_, _ = fmt.Fprintln(os.Stderr, append([]any{f}, args...))
		return
	}
	var buf = new(bytes.Buffer)
	_, _ = fmt.Fprintf(buf, format, args...)
	message := buf.String()
	for len(message) > 0 {
		if !strings.ContainsRune("\n\r\t ", rune(message[len(message)-1])) {
			break
		}
		message = message[:len(message)-1]
	}
	_, _ = fmt.Fprintln(os.Stderr, message)
}

type deadlineListener interface {
	net.Listener
	SetDeadline(time.Time) error
}

// Similar to ssh.DiscardRequests but less prone to goroutine leaks,
// this function exits on context cancellation and on closing requests channel.
func discard(ctx context.Context, reqs <-chan *ssh.Request) {
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
			var allow bool
			switch r.Type {
			case "shell", "pty-req":
				allow = true
			default:
				allow = false
			}
			if r.WantReply {
				_ = r.Reply(allow, nil)
			}
		}
	}
}

func reject(ctx context.Context, chans <-chan ssh.NewChannel) {
	for {
		select {
		case <-ctx.Done():
			return
		case ch, ok := <-chans:
			if !ok {
				return
			}
			if ch == nil {
				continue
			}
			_ = ch.Reject(ssh.Prohibited, "only one channel per connection is expected")
		}
	}
}
