package pid1

import (
	"fmt"
	"os"
	"syscall"
)

// Mount /dev, /sys, /proc filesystems
func mountDevSysProc() error {
	for _, m := range []struct {
		source string
		target string
		fstype string
	}{
		{"none", "/proc", "proc"},
		{"none", "/sys", "sysfs"},
		{"none", "/dev", "devtmpfs"},
	} {
		err := mount(m.source, m.target, m.fstype)
		if err != nil {
			return err
		}
	}
	return nil
}

func mount(source, target, fstype string) error {
	var err error
	err = os.MkdirAll(target, 0755)
	if err != nil {
		return err
	}

	err = syscall.Mount(source, target, fstype, 0, "")
	sce, ok := err.(syscall.Errno)
	if err != nil && ok && sce != syscall.EBUSY { // EBUSY means device is already mounted there
		return fmt.Errorf("%s: %w", target, err)
	}
	return nil
}
