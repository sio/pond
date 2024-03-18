package pid1

import (
	"fmt"
	"math/rand"
	"sync"
	"syscall"
	"time"
)

// Default init target
var Target Task = "Switch root"

var tasks = map[Task]*run{
	"Mount /dev, /sys, /proc": &run{
		Do: mountDevSysProc,
	},
	"Load kernel modules": &run{
		Do: loadDeviceModules,
		After: []Task{
			"Mount /dev, /sys, /proc",
		},
	},
	"Bring up the network": &run{
		Do: networkUp,
		After: []Task{
			"Load kernel modules",
		},
	},
	"Switch root": &run{}, // TODO: implement the last task
}

// Execute init process with default target
func Run() {
	// TODO: remove dummy tasks from initramfs
	wait := func() error {
		n := time.Duration(rand.Intn(10) + 1)
		time.Sleep(time.Second * n / 100)
		return nil
	}
	fail := func() error {
		return fmt.Errorf("failed")
	}
	NewTask("foo", wait)
	NewTask("bar", wait)
	NewTask("baz", wait, "foo", "bar")
	NewTask("Expect failure", fail)

	// TODO: remove temporary shell from initramfs
	shell := func() error {
		return syscall.Exec(
			"/bin/setsid",
			[]string{"setsid", "sh", "-c", "exec sh </dev/ttyS0 >/dev/ttyS0 2>&1"},
			nil,
		)
	}
	NewTaskConfig(
		"Debug shell",
		shell,
		[]Task{
			"baz",
			"Bring up the network",
		},
		[]Task{Target},
	)

	RunUntil(Target)
}

// Execute init process until a specific target is reached
func RunUntil(t Task) {
	q := NewTaskQueue()
	go q.PrintResults()
	start(t, q)
	q.Wait(t)
}

func start(t Task, q *TaskQueue) {
	job, ok := tasks[t]
	if !ok {
		panic("invalid target task: " + t)
	}
	if job.Started {
		return
	}
	for _, dep := range job.After {
		start(dep, q)
	}
	tasksMu.Lock()
	job.Started = true
	q.Go(t, job.Do, job.After...)
	tasksMu.Unlock()
}

var tasksMu sync.Mutex

// Add a new task to init dependency tree
func NewTask(t Task, do func() error, after ...Task) {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	tasks[t] = &run{
		Do:    do,
		After: after,
	}
}

// Add a new task to init dependency tree - more tunable knobs
func NewTaskConfig(t Task, do func() error, after []Task, before []Task) {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	for _, next := range before {
		job, ok := tasks[next]
		if !ok {
			panic("invalid task name: " + next)
		}
		job.After = append(job.After, t)
	}
	tasks[t] = &run{
		Do:    do,
		After: after,
	}
}

type run struct {
	Do      func() error
	After   []Task
	Started bool
}
