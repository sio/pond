package server

import (
	"encoding/binary"
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

	nbd := &nbdConn{Conn: conn}
	defer nbd.Close()

	err := nbd.Handshake()
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("Success: %v", conn)
}

// Narrow purpose NBD connection handler
type nbdConn struct {
	net.Conn
	tls     bool
	backend io.ReaderAt
}

func (nbd *nbdConn) Close() {
	if nbd.Conn != nil {
		_ = nbd.Conn.Close()
	}
}

func (nbd *nbdConn) Errorf(format string, args ...any) error {
	addr := nbd.RemoteAddr()
	prefix := fmt.Sprintf("%s://%s: ", addr.Network(), addr.String())
	return fmt.Errorf(prefix+format, args...)
}

func (nbd *nbdConn) Handshake() error {
	err := binary.Write(nbd, binary.BigEndian, nbdHandshakeServer)
	if err != nil {
		return nbd.Errorf("handshake: %w", err)
	}
	var reply nbdHandshakeClient
	err = binary.Read(nbd, binary.BigEndian, &reply)
	if err != nil {
		return nbd.Errorf("handshake: %w", err)
	}
	if reply.Padding != 0 {
		return nbd.Errorf("handshake: client reply: unexpected flags instead of padding: %016b", reply.Padding)
	}
	if reply.Flag != nbdHandshakeServer.flag {
		return nbd.Errorf("handshake: client reply: unexpected flags: %016b", reply.Flag)
	}
	return nil
}

func (s *Server) nbdTransmission(conn net.Conn, backend io.ReaderAt) {
	if backend == nil {
		return
	}
}

var nbdHandshakeServer = struct {
	magic  uint64
	option uint64
	flag   handshakeFlag
}{
	magic:  NBDMAGIC,
	option: IHAVEOPT,
	flag:   NBD_FLAG_FIXED_NEWSTYLE,
}

type nbdHandshakeClient struct {
	Padding uint16
	Flag    handshakeFlag
}

func (r nbdHandshakeClient) String() string {
	return fmt.Sprintf("%x %x (%016b %016b)", r.Padding, r.Flag, r.Padding, r.Flag)
}

type nbdNegotiationQuery struct {
}

type nbdNegotiationReply struct {
	magic  uint64
	option optionType
	reply  replyType
	size   uint32
	data   *[]byte
}
