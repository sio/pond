package pre

import (
	"fmt"
	"strings"
	"sync"
)

type Task string

type TaskQueue struct {
	tasks   map[Task]chan struct{}
	mu      sync.Mutex
	results chan TaskResult
}

type TaskResult struct {
	Task Task
	Err  error
}

func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks:   make(map[Task]chan struct{}),
		results: make(chan TaskResult, 16),
	}
}

func (q *TaskQueue) Wait(tasks ...Task) {
	for _, task := range tasks {
		ch := q.ch(task)
		<-ch
	}
}

func (q *TaskQueue) Go(task Task, worker func() error, depends ...Task) {
	q.Wait(depends...)
	go q.do(task, worker)
}

func (q *TaskQueue) do(task Task, worker func() error) {
	err := worker()
	q.results <- TaskResult{task, err}
	if err == nil {
		q.done(task)
	}
}

func (q *TaskQueue) done(task Task) {
	close(q.ch(task))
}

func (q *TaskQueue) ch(task Task) chan struct{} {
	ch, exists := q.tasks[task]
	if !exists {
		q.mu.Lock()
		defer q.mu.Unlock()
		ch = make(chan struct{})
		q.tasks[task] = ch
	}
	return ch
}

func (q *TaskQueue) Status() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	waiting := make([]string, 0, len(q.tasks))
	complete := 0
	for task, ch := range q.tasks {
		select {
		case <-ch:
			complete++
		default:
			waiting = append(waiting, string(task))
		}
	}
	if len(waiting) > 0 {
		return fmt.Sprintf("Blocked. %d tasks complete. Waiting for %s", complete, strings.Join(waiting, ", "))
	} else {
		return fmt.Sprintf("Not blocked. %d tasks complete.", complete)
	}
}

const (
	asciiEsc   = "\u001B["
	asciiRed   = asciiEsc + "31;1m"
	asciiGreen = asciiEsc + "32;1m"
	asciiReset = asciiEsc + "0m"
	tagError   = asciiRed + " FAIL " + asciiReset
	tagOK      = asciiGreen + "  OK  " + asciiReset
)

func (q *TaskQueue) PrintResults() {
	var status, tag string
	for r := range q.results {
		status = "done"
		tag = tagOK
		if r.Err != nil {
			tag = tagError
			status = r.Err.Error()
		}
		fmt.Printf("[%s] %s... %s.\n", tag, r.Task, status)
	}
}
