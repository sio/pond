package pid1

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/sio/pond/initramfs/rand"
	"golang.org/x/sys/unix"
)

// Add entropy to system random number generator
func seedRandomGenerator() error {
	urandom, err := os.OpenFile("/dev/urandom", os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = urandom.Close() }()
	entropy := struct {
		bits int64
		size int64
		buf  [512]byte
	}{}
	rand.Seed(entropy.buf[:])
	entropy.size = int64(len(entropy.buf))
	entropy.bits = entropy.size * 8 // this is a severe overestimation, our RNG output does not contain 1:1 entropy

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
