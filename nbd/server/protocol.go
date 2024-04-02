package server

import (
	"fmt"
	"io"
	"log"
	"net"
)

// Speak NBD protocol over a single TCP/TLS connection.
//
// Unlike with other common layer 7 protocols (like HTTP) these connections are
// very long lived.
func (s *Server) handleConnection(conn net.Conn) {
	s.wg.Add(1)
	defer s.wg.Done()
	defer conn.Close()

	nbd, err := s.newConnection(conn)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Negotiation OK:", nbd)
}

func (s *Server) newConnection(conn net.Conn) (*nbdConn, error) {
	nbd := &nbdConn{Conn: conn}
	err := handshake(nbd)
	if err != nil {
		return nil, nbd.Errorf("handshake: %w", err)
	}
	nbd.backend, err = negotiate(s.ctxSoft, nbd, s.export)
	if err != nil {
		return nil, nbd.Errorf("negotiation: %w", err)
	}
	return nbd, nil
}

// Narrow purpose NBD connection handler
type nbdConn struct {
	net.Conn
	tls     bool
	backend Backend
}

func (nbd *nbdConn) Close() {
	if nbd.Conn != nil {
		_ = nbd.Conn.Close()
	}
	backend, ok := nbd.backend.(io.Closer)
	if ok {
		_ = backend.Close()
	}
}

func (nbd *nbdConn) Error(e error) error {
	if e == nil {
		return nil
	}
	return nbd.Errorf("%w", e)
}

func (nbd *nbdConn) Errorf(format string, args ...any) error {
	addr := nbd.RemoteAddr()
	prefix := fmt.Sprintf("%s://%s: ", addr.Network(), addr.String())
	return fmt.Errorf(prefix+format, args...)
}
