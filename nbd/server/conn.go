package server

import (
	"fmt"
	"io"
	"net"
)

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
