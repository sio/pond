package pid1

import (
	"fmt"
	"math/rand"
	"syscall"
	"time"
)

func Run() {
	task := NewTaskQueue()
	go task.PrintResults()
	wait := func() error {
		n := time.Duration(rand.Intn(10) + 1)
		time.Sleep(time.Second * n / 100)
		fmt.Println(task.Status())
		return nil
	}
	fail := func() error {
		return fmt.Errorf("failed")
	}
	task.Go("foo", wait)
	task.Go("bar", wait)
	task.Go("baz", wait, "foo", "bar")
	task.Go("Expect failure", fail)
	task.Wait("baz")
	fmt.Println(task.Status())

	// TODO: remove temporary shell from initramfs
	err := syscall.Exec("/bin/sh", []string{"sh"}, nil)
	if err != nil {
		panic(err)
	}

	// PID 1 (init process) must never exit, this would lead to kernel panic.
	// We expect to switch_root into full rootfs eventually.
	// Deadlock may occur at this stage only if an essential PID 1 task fails.
	select {}
}
