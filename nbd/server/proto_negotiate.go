package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/sio/pond/nbd/buffer"
)

// Send arbitrary objects over given connection
func send(conn io.Writer, from ...any) error {
	for _, data := range from {
		if data == nil {
			continue
		}
		err := binary.Write(conn, binary.BigEndian, data)
		if err != nil {
			return err
		}
	}
	return nil
}

// Receive arbitrary objects from given connection
func receive(conn io.Reader, into ...any) error {
	for _, data := range into {
		err := binary.Read(conn, binary.BigEndian, data)
		if err != nil {
			return err
		}
	}
	return nil
}

// NBD Handshake Phase
func handshake(conn io.ReadWriter) error {
	var hello = struct {
		magic  uint64
		option uint64
		flag   handshakeFlag
	}{
		magic:  NBDMAGIC,
		option: IHAVEOPT,
		flag:   NBD_FLAG_FIXED_NEWSTYLE,
	}
	err := send(conn, hello)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	var reply struct {
		Padding uint16
		Flag    handshakeFlag
	}
	err = receive(conn, &reply)
	if err != nil {
		return fmt.Errorf("receive: %w", err)
	}
	if reply.Padding != 0 {
		return fmt.Errorf("client reply: unexpected flags instead of padding: %016b", reply.Padding)
	}
	if reply.Flag != hello.flag {
		return fmt.Errorf("client reply: unexpected flags: %016b", reply.Flag)
	}
	return nil
}

// NBD Negotiation Phase
func negotiate(ctx context.Context, conn io.ReadWriter, export func(name string) (Backend, error)) (Backend, error) {
	reply := func(t optionType, r optionReply, data ...any) error {
		var d []byte
		if len(data) != 0 {
			buf := buffer.Get()
			defer buffer.Put(buf)
			payload := &byteWriter{buf: buf}
			err := send(payload, data...)
			if err != nil {
				return fmt.Errorf("serializing payload: %w", err)
			}
			d = payload.Bytes()
		}
		return send(
			conn,
			struct {
				Magic uint64
				Type  optionType
				Reply optionReply
				Len   uint32
			}{
				Magic: 0x3e889045565a9,
				Type:  t,
				Reply: r,
				Len:   uint32(len(d)),
			},
			d,
		)
	}

	for {
		var option struct {
			Magic uint64
			Type  optionType
			Len   uint32
		}
		err := receive(conn, &option)
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("client terminated connection")
		}
		if err != nil {
			return nil, fmt.Errorf("receiving option: %w", err)
		}
		if option.Magic != IHAVEOPT {
			return nil, fmt.Errorf("bad IHAVEOPT from client: %x", option.Magic)
		}
		select {
		case <-ctx.Done():
			_ = reply(option.Type, NBD_REP_ERR_SHUTDOWN, nil)
			return nil, context.Cause(ctx)
		default:
		}

		switch option.Type {

		case NBD_OPT_GO, NBD_OPT_INFO:
			if option.Len > buffer.Size {
				err = discard(conn, int(option.Len))
				if err != nil {
					return nil, err
				}
				err = reply(option.Type, NBD_REP_ERR_TOO_BIG, nil)
				if err != nil {
					return nil, err
				}
				continue
			}
			backend, err := negotiateBackend(conn, export, option.Len)
			if err != nil {
				ereply := reply(option.Type, NBD_REP_ERR_UNKNOWN, []byte("requested export is not available\x00"))
				if ereply != nil {
					return nil, ereply
				}
				return nil, err
			}

			// Ignore all information requests sent by client,
			// always send the same set of information replies.
			err = reply(option.Type, NBD_REP_INFO, struct {
				info infoType
				size uint64
				flag transmissionFlag
			}{
				info: NBD_INFO_EXPORT,
				flag: NBD_FLAG_HAS_FLAGS |
					NBD_FLAG_READ_ONLY |
					NBD_FLAG_CAN_MULTI_CONN |
					NBD_FLAG_SEND_CACHE,
				// Use an obviously bogus number for export size to make sure
				// no one confuses it for a real one.
				// Value of one exabyte also shows up nicely as 1E in lsblk
				// hinting that it might be an (E)rror.
				size: 1 << 60, // TODO: pass real export size here somehow?
			})
			if err != nil {
				return nil, fmt.Errorf("NBD_INFO_EXPORT: %w", err)
			}
			// TODO: send NBD_INFO_BLOCK_SIZE with prefered block size = BufferSize

			// Finish successfully
			err = reply(option.Type, NBD_REP_ACK, nil)
			if err != nil {
				return nil, err
			}
			if option.Type == NBD_OPT_GO {
				return backend, nil
			}

		case NBD_OPT_EXPORT_NAME: // not supported; drop connection (violates NBD protocol spec)
			_ = reply(option.Type, NBD_REP_ERR_POLICY, []byte("this server requires fixed newstyle negotiation\x00"))
			return nil, fmt.Errorf("client attempted non-fixed newstyle negotiation")

		case NBD_OPT_ABORT:
			_ = reply(option.Type, NBD_REP_ACK, nil)
			return nil, fmt.Errorf("client desired to end the negotiation")

		default: // all other options are not supported; ignore
			err = discard(conn, int(option.Len))
			if err != nil {
				return nil, err
			}
			err = reply(option.Type, NBD_REP_ERR_UNSUP, nil)
			if err != nil {
				return nil, err
			}
		}
	}
}

// Negotiate NBD export with client
func negotiateBackend(conn io.ReadWriter, export func(name string) (Backend, error), size uint32) (Backend, error) {
	buf := buffer.Get()
	defer buffer.Put(buf)

	payloadLen := int(size)
	if payloadLen > cap(buf) {
		_ = discard(conn, payloadLen)
		return nil, fmt.Errorf("payload too large: %db > %db", size, cap(buf))
	}
	if payloadLen < 4+2 {
		_ = discard(conn, payloadLen)
		return nil, fmt.Errorf("payload too small: %db", size)
	}
	buf = buf[:payloadLen]
	err := receive(conn, buf)
	if err != nil {
		return nil, fmt.Errorf("reading payload: %w", err)
	}
	payload := bytes.NewReader(buf)
	var nameLen uint32
	err = receive(payload, &nameLen)
	if err != nil {
		return nil, fmt.Errorf("reading export name length: %w", err)
	}
	_, err = payload.Seek(int64(nameLen), io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("can not parse export name, payload too short")
	}
	if export == nil {
		return nil, fmt.Errorf("no exports defined for this server")
	}
	backend, err := export(string(buf[4 : 4+int(nameLen)]))
	if err != nil {
		return nil, fmt.Errorf("export not available: %w", err)
	}
	return backend, err
}
