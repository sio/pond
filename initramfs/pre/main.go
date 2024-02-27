package pre

import (
	"fmt"
	"math/rand"
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
}
