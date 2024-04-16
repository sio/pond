package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/sio/pond/nbd/buffer"
)

// NBD Transmission Phase
func transmission(ctx context.Context, conn io.ReadWriter, backend Backend) error {
	var request sync.WaitGroup
	defer request.Wait()

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	var write sync.Mutex

	sendError := func(cookie clientCookie, err nbdError) error {
		write.Lock()
		defer write.Unlock()
		return send(conn, replyHeader{
			Magic:  NBD_SIMPLE_REPLY_MAGIC,
			Error:  err,
			Cookie: cookie,
		})
	}

	commands := make(chan requestHeader)
	go func() {
		var cmd requestHeader
		for {
			err := receive(conn, &cmd)
			if err != nil {
				cancel(fmt.Errorf("receive command: %w", err))
				return
			}
			select {
			case commands <- cmd:
				// continue to receive next command
			case <-ctx.Done():
				return
			}
		}
	}()
	defer log.Println("Exit transmission:", conn)

	var err error
	for {
		var cmd requestHeader
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case cmd = <-commands:
		}
		if cmd.Magic != NBD_REQUEST_MAGIC {
			return fmt.Errorf("invalid NBD_REQUEST_MAGIC: got %x, want %x", cmd.Magic, NBD_REQUEST_MAGIC)
		}
		if cmd.Flag != 0 {
			// Our server does not support any NBD_CMD_FLAG_*
			err = sendError(cmd.Cookie, NBD_EINVAL)
			if err != nil {
				return fmt.Errorf("error while rejecting unsupported command flags (%d): %w", cmd.Flag, err)
			}
			continue
		}

		switch cmd.Type {

		case NBD_CMD_READ:
			request.Add(1)
			go func(cmd requestHeader) {
				//log.Printf("read request %#x+%db", cmd.Offset, cmd.Len)
				defer request.Done()

				buf := buffer.Get()
				defer buffer.Put(buf)
				buf = buf[:cap(buf)]

				var inTransaction bool
				var ioErrors int

				cur := int64(cmd.Offset)
				end := cur + int64(cmd.Len)
				for cur < end {
					if (end - cur) < int64(len(buf)) {
						buf = buf[:int(end-cur)]
					}
					n, ioerr := backend.ReadAt(buf, cur)
					/**

					TRUTH TABLE FOR WHAT HAPPENS NEXT

					.-------.---------------.------.-----------------------.
					| ioerr | inTransaction |  n   |    (Order) Action     |
					:-------+---------------+------+-----------------------:
					| error | no            | 0    | (1) NBD_EIO           |
					:-------+---------------+------+-----------------------:
					| error | yes           | 0    | (2) ioErrors++        |
					:-------+---------------+------+-----------------------:
					| nil   | no            | some | (3) Begin transaction |
					:-------+---------------+------+-----------------------:
					| error | no            | some | (3) Begin transaction |
					:-------+---------------+------+-----------------------:
					| nil   | yes           | some | (4) Send data         |
					:-------+---------------+------+-----------------------:
					| error | yes           | some | (4) Send data         |
					:-------+---------------+------+-----------------------:
					| nil   | no            | 0    | ( ) noop              |
					:-------+---------------+------+-----------------------:
					| nil   | yes           | 0    | ( ) noop              |
					'-------'---------------'------'-----------------------'
					**/
					if ioerr == nil {
						ioErrors = 0 // reset error counter
					}

					// (1) NBD_EIO
					if ioerr != nil && !inTransaction && n == 0 {
						err := sendError(cmd.Cookie, NBD_EIO)
						if err != nil {
							cancel(fmt.Errorf("backend error (%w) followed by connection error (%w)", ioerr, err))
						}
						return
					}

					// (2) ioErrors++
					if ioerr != nil && inTransaction && n == 0 {
						ioErrors++
						const ioErrorsThreshold = 30
						if ioErrors > ioErrorsThreshold {
							// Violates NBD protocol spec!
							//
							// Blatantly lie to client because it's already too late to
							// error out of NBD_CMD_READ (we have sent reply header with
							// error=0).
							//
							// Structured replies were created to help in such scenario,
							// but unfortunately they are still not supported by reference
							// client implementation and by Linux kernel.
							//
							// We rely on data integrity verification being implemented on
							// top of our block device (dm-verity, zfs, btrfs)
							for n < len(buf) {
								n += copy(buf[n:], bogusData[:])
							}
						}
					}

					// (3) Begin transaction
					if !inTransaction && n != 0 {
						write.Lock()
						defer write.Unlock()
						err := send(conn, replyHeader{
							Magic:  NBD_SIMPLE_REPLY_MAGIC,
							Error:  0,
							Cookie: cmd.Cookie,
						})
						if err != nil {
							cancel(fmt.Errorf("NBD_CMD_READ: begin transaction: %w", err))
							return
						}
						inTransaction = true
					}

					// (4) Send data
					cur += int64(n)
					err = send(conn, buf[:n])
					if err != nil {
						cancel(fmt.Errorf("NBD_CMD_READ: send data: %w", err))
						return
					}
				}
			}(cmd)

		// case NBD_CMD_CACHE: // TODO: handle cache requests

		case NBD_CMD_DISC: // Disconnect
			return nil

		default: // Other commands are not supported
			if cmd.Type == NBD_CMD_WRITE {
				err = discard(conn, int(cmd.Len))
				if err != nil {
					return fmt.Errorf("discarding command (%v) payload: %w", cmd.Type, err)
				}
			}
			err = sendError(cmd.Cookie, NBD_ENOTSUP)
			if err != nil {
				return fmt.Errorf("rejecting unsupported command (%v): %w", cmd.Type, err)
			}
		}
	}
}

type clientCookie uint64

type requestHeader struct {
	Magic  uint32
	Flag   requestFlag
	Type   requestType
	Cookie clientCookie
	Offset uint64
	Len    uint32
}

type requestFlag uint16

type replyHeader struct {
	Magic  uint32
	Error  nbdError
	Cookie clientCookie
}

// Searchable bogus data (0xEEE0E1E2E3E4E5E6E7E8E9EAEBECEDEE in a loop) that we
// send to client when we have no other choice other than this (violates NBD protocol spec)
// or severing TCP connection.
var bogusData = [...]byte{0xee, 0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee}
