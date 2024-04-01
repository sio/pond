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

	err := nbd.Handshake()
	if err != nil {
		return nil, err
	}

	allow_structured_reply := false
	for {
		option, err := nbd.readOptionHeader()
		if err != nil {
			return nil, err
		}
		if s.shutdown {
			_ = nbd.reply(option.Type, NBD_REP_ERR_SHUTDOWN, nil)
			return nil, NBD_ESHUTDOWN
		}
		log.Printf("got option: %v", option)
		switch option.Type {

		case NBD_OPT_GO:
			if !allow_structured_reply {
				// TODO: require structured replies
			}
			err = s.info(nbd, option)
			if err != nil {
				return nil, err
			}
			return nbd, nil // proceed to transmission phase

		case NBD_OPT_INFO:
			err = s.info(nbd, option)
			if err != nil {
				return nil, err
			}

		case NBD_OPT_EXPORT_NAME: // intentionally not supported; drop connection
			_ = nbd.reply(option.Type, NBD_REP_ERR_UNSUP, nil)
			return nil, nbd.Errorf("client attempted non-fixed newstyle negotiation")

		case NBD_OPT_ABORT:
			_ = nbd.reply(option.Type, NBD_REP_ACK, nil)
			return nil, nbd.Errorf("client desired to end the negotiation")

		case NBD_OPT_STRUCTURED_REPLY:
			if option.Len != 0 {
				err = nbd.reply(option.Type, NBD_REP_ERR_INVALID, nil)
				if err != nil {
					return nil, err
				}
				err = nbd.Discard(option.Len)
				if err != nil {
					return nil, err
				}
				continue
			}
			allow_structured_reply = true
			err = nbd.reply(option.Type, NBD_REP_ACK, nil)
			if err != nil {
				return nil, err
			}

		default: // all other options are not supported; ignore
			err = nbd.Discard(option.Len)
			if err != nil {
				return nil, err
			}
			err = nbd.reply(option.Type, NBD_REP_ERR_UNSUP, nil)
			if err != nil {
				return nil, err
			}
		}
	}
}

func (s *Server) info(nbd *nbdConn, info optionHeader) error {
	err := s.selectExport(nbd, info)
	if err != nil {
		_ = nbd.reply(info.Type, NBD_REP_ERR_INVALID, nil) // TODO: this is out of spec
		return nbd.Errorf("%w", err)
	}
	return nil
}

func (s *Server) selectExport(nbd *nbdConn, info optionHeader) error {
	buf := buffer.Get().([]byte)
	defer buffer.Put(buf)

	payloadLen := int(info.Len)
	if payloadLen > cap(buf) {
		_ = nbd.Discard(info.Len)
		return fmt.Errorf("payload too large: %d bytes", payloadLen)
	}
	if payloadLen < 4+2 {
		_ = nbd.Discard(info.Len)
		return fmt.Errorf("payload too short: %d bytes", payloadLen)
	}
	buf = buf[:payloadLen]
	err := nbd.Read(buf)
	if err != nil {
		return err
	}
	var nameLen uint32
	for i := 0; i < 4; i++ {
		nameLen <<= 8
		nameLen |= uint32(buf[i])
	}
	if payloadLen < 4+2+int(nameLen) {
		return fmt.Errorf("export name size (%db) does not fit into payload (%db)", nameLen, payloadLen)
	}
	var requestLen uint16
	for i := 0; i < 2; i++ {
		requestLen <<= 8
		requestLen |= uint16(buf[4+int(nameLen)+i])
	}
	if payloadLen != 4+int(nameLen)+2+int(requestLen)*2 {
		return fmt.Errorf("malformed payload (%db) (n=%d, r=%d, b=%x)", payloadLen, nameLen, requestLen, buf)
	}
	if s.export == nil {
		return fmt.Errorf("no exports defined for this server")
	}
	backend, err := s.export(string(buf[4 : 4+nameLen]))
	if err != nil {
		return err
	}
	// TODO: send NBD_INFO_EXPORT, NBD_INFO_BLOCK_SIZE
	err = nbd.Write(struct { // TODO: forgot protocol header before payload
		tag  uint16
		size uint64
		flag transmissionFlag
	}{
		tag:  0,                  // NBD_INFO_EXPORT
		size: 0xffffffffffffffff, // TODO: pass real export size here somehow?
		flag: NBD_FLAG_HAS_FLAGS | NBD_FLAG_READ_ONLY | NBD_FLAG_CAN_MULTI_CONN | NBD_FLAG_SEND_CACHE,
	})
	if info.Type == NBD_OPT_GO {
		nbd.backend = backend
	}
	return nil
}

// Narrow purpose NBD connection handler
type nbdConn struct {
	net.Conn
	tls     bool
	backend io.ReaderAt
}

func (nbd *nbdConn) Read(into any) error {
	return binary.Read(nbd.Conn, binary.BigEndian, into)
}

func (nbd *nbdConn) Write(from any) error {
	return binary.Write(nbd.Conn, binary.BigEndian, from)
}

func (nbd *nbdConn) Discard(n uint32) error {
	return discard(nbd.Conn, int(n))
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
	err := nbd.Write(nbdHandshakeServer)
	if err != nil {
		return nbd.Errorf("handshake: %w", err)
	}
	var reply nbdHandshakeClient
	err = nbd.Read(&reply)
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

func (nbd *nbdConn) readOptionHeader() (header optionHeader, err error) {
	err = nbd.Read(&header)
	if err != nil {
		return header, nbd.Errorf("negotiating options: %w", err)
	}
	if header.Magic != IHAVEOPT {
		return header, nbd.Errorf("negotiating options: bad IHAVEOPT from client: %x", header.Magic)
	}
	return header, nil
}

func (nbd *nbdConn) reply(t optionType, r optionReply, data []byte) error {
	err := nbd.Write(optionReplyHeader{
		Magic: 0x3e889045565a9,
		Type:  t,
		Reply: r,
		Len:   uint32(len(data)),
	})
	if err != nil {
		return err
	}
	return nbd.Write(data)
}

type optionHeader struct {
	Magic uint64
	Type  optionType
	Len   uint32
}

type optionReplyHeader struct {
	Magic uint64
	Type  optionType
	Reply optionReply
	Len   uint32
}
