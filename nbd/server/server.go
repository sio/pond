// Read-only NBD server
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
	"time"
)

func New(ctx context.Context, export func(name string) (io.ReaderAt, error)) *Server {
	ctx, cancel := context.WithCancel(ctx)
	return &Server{
		export: export,
		ctx:    ctx,
		cancel: cancel,
	}
}

const (
	gracefulShutdownTimeout = 30 * time.Second
	connAcceptTimeout       = gracefulShutdownTimeout / 10
)

type Server struct {
	export   func(name string) (io.ReaderAt, error)
	ctx      context.Context
	cancel   context.CancelFunc
	shutdown bool
	wg       sync.WaitGroup
	err      error
}

// Listen for incoming NBD connections indefinitely
func (s *Server) Listen(network, address string) error {
	tcp := &net.ListenConfig{}
	l, err := tcp.Listen(s.ctx, network, address)
	if err != nil {
		return err
	}
	defer func() { _ = l.Close() }()
	listener, ok := l.(deadlineListener)
	if !ok {
		return fmt.Errorf("%T does not support deadline", l)
	}
	for {
		if s.shutdown {
			return s.err
		}
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
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
		return
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig...)
	for interrupt := range ch {
		log.Printf("Initiating graceful shutdown due to %s", interrupt)
		s.Shutdown()
	}
}

// Shutdown server gracefully
func (s *Server) Shutdown() {
	go func() {
		select {
		case <-s.ctx.Done():
		case <-time.After(gracefulShutdownTimeout):
			log.Fatalf("Graceful shutdown took longer than %s, crashing hard", gracefulShutdownTimeout)
		}
	}()
	s.shutdown = true // do not accept new connections and commands
	s.wg.Wait()       // handle all outstanding requests
	s.cancel()        // drop everything
}

type deadlineListener interface {
	net.Listener
	SetDeadline(time.Time) error
}
