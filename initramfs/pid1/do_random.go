package pid1

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Use current time to add entropy to system random number generator
func seedRandomGenerator() error {
	urandom, err := os.OpenFile("/dev/urandom", os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = urandom.Close() }()
	seed := new(bytes.Buffer)
	err = binary.Write(seed, binary.BigEndian, time.Now().UnixNano())
	if err != nil {
		return err
	}
	_, err = seed.WriteString(time.Now().String())
	if err != nil {
		return err
	}
	err = binary.Write(seed, binary.LittleEndian, time.Now().UnixNano())
	if err != nil {
		return err
	}
	entropy := struct {
		bits int64
		size int64
		buf  [512]byte
	}{
		bits: int64(seed.Len()) * 8,
		size: int64(seed.Len()),
	}
	copy(entropy.buf[:], seed.Bytes())

	const RNDADDENTROPY = 0x40085203
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		urandom.Fd(),
		RNDADDENTROPY,
		uintptr(unsafe.Pointer(&entropy)),
	)
	if errno != 0 {
		return fmt.Errorf("RNDADDENTROPY: %w", errno)
	}
	return nil

}
