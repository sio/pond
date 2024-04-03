// Read-only NBD server
//
// This server was implemented for a very narrow usage scenario.
// Be careful when attempting to use it outside of pond/nbd project: this
// implementation omits several key features of NBD protocol spec, violates
// multiple protocol requirements and even sends bogus data to client in some
// cases.
//
// You have been warned!
package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Actual storage interaction happens through this object
type Backend = io.ReaderAt

func New(ctx context.Context, export func(name string) (Backend, error)) *Server {
	s := &Server{export: export}
	s.ctxStrict, s.cancelStrict = context.WithCancelCause(ctx)
	s.ctxSoft, s.cancelSoft = context.WithCancelCause(s.ctxStrict)
	return s
}

const (
	gracefulShutdownTimeout = 30 * time.Second
	connAcceptTimeout       = gracefulShutdownTimeout / 10
)

type Server struct {
	export                   func(name string) (Backend, error)
	ctxSoft, ctxStrict       context.Context
	cancelSoft, cancelStrict context.CancelCauseFunc
	conn                     sync.WaitGroup
}

// Listen for incoming NBD connections indefinitely
func (s *Server) Listen(network, address string) error {
	tcp := &net.ListenConfig{}
	l, err := tcp.Listen(s.ctxStrict, network, address)
	if err != nil {
		return err
	}
	defer func() { _ = l.Close() }()
	listener, ok := l.(deadlineListener)
	if !ok {
		return fmt.Errorf("%T does not support deadline", l)
	}
	for {
		select {
		case <-s.ctxSoft.Done():
			return context.Cause(s.ctxSoft)
		case <-s.ctxStrict.Done():
			return context.Cause(s.ctxStrict)
		default:
		}
		err = listener.SetDeadline(time.Now().Add(connAcceptTimeout))
		if err != nil {
			return err
		}
		conn, err := listener.Accept()
		if os.IsTimeout(err) {
			continue
		}
		if err != nil {
			log.Println(err)
			continue
		}
		go s.handleConnection(conn)
	}
}

// Listen for OS signals to initiate graceful shutdown
func (s *Server) ListenShutdown(sig ...os.Signal) {
	if len(sig) == 0 {
		sig = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig...)
	for interrupt := range ch {
		log.Printf("Initiating graceful shutdown due to %s", interrupt)
		s.Shutdown()
		return
	}
}

// Shutdown server gracefully
func (s *Server) Shutdown() {
	go func() {
		select {
		case <-s.ctxStrict.Done():
		case <-time.After(gracefulShutdownTimeout):
			s.cancelStrict(NBD_ESHUTDOWN)
			time.Sleep(time.Second)
			log.Fatalf("Graceful shutdown took longer than %s, crashing hard", gracefulShutdownTimeout)
		}
	}()
	s.cancelSoft(NBD_ESHUTDOWN)   // do not accept new connections and commands
	s.conn.Wait()                 // handle all established client connections
	s.cancelStrict(NBD_ESHUTDOWN) // drop everything
}

type deadlineListener interface {
	net.Listener
	SetDeadline(time.Time) error
}

// Service a single client connection.
//
// Unlike with other common layer 7 protocols (like HTTP) these connections are
// very long lived.
func (s *Server) handleConnection(conn net.Conn) {
	s.conn.Add(1)
	defer s.conn.Done()
	defer func() { _ = conn.Close() }()

	addr := conn.RemoteAddr()
	client := fmt.Sprintf("%s://%s: ", addr.Network(), addr.String())

	err := s.serveNBD(conn)
	if err != nil {
		log.Printf("%s: disconnect on failure: %v", client, err)
		return
	}
	log.Printf("%s: disconnect on success", client)
}

// Speak NBD protocol over a single TCP/TLS connection
func (s *Server) serveNBD(conn net.Conn) error {
	err := handshake(conn)
	if err != nil {
		return fmt.Errorf("handshake: %w", err)
	}
	backend, err := negotiate(s.ctxSoft, conn, s.export)
	if err != nil {
		return fmt.Errorf("negotiation: %w", err)
	}
	if b, ok := backend.(io.Closer); ok {
		defer func() { _ = b.Close() }()
	}
	err = transmission(s.ctxSoft, conn, backend)
	if err != nil {
		return fmt.Errorf("transmission: %w", err)
	}
	return nil
}
