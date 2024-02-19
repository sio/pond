package pre

import (
	"fmt"
	"time"
)

func Run() {
	task := &TaskManager{}
	go task.Wait("foo", "bar")
	go task.Wait("baz")
	for _, name := range []string{"hello", "world", "foo", "bar", "baz", "eh"} {
		time.Sleep(time.Second / 10)
		fmt.Println(task.Status())
		task.Done(name)
	}
	task.Wait()
}
