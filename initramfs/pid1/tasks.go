package pid1

import (
	"fmt"
	"strings"
	"sync"
)

// Human readable name for init task
type Task string

// A task queue that makes good use of Go concurrency model to launch init
// tasks as fast as possible
type TaskQueue struct {
	tasks   map[Task]chan struct{}
	mu      sync.Mutex
	results chan result
}

type result struct {
	Task Task
	Err  error
}

func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks:   make(map[Task]chan struct{}),
		results: make(chan result, 16),
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
	ch := q.ch(task)
	select {
	case <-ch:
		return // do not do the same task again
	default:
	}
	go q.do(task, worker)
}

func (q *TaskQueue) do(task Task, worker func() error) {
	err := worker()
	q.results <- result{task, err}
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

func (q *TaskQueue) PrintResults() {
	var status string
	var log func(f string, a ...any)
	for r := range q.results {
		status = "done"
		log = MsgOK
		if r.Err != nil {
			status = r.Err.Error()
			log = MsgErr
		}
		log("%s... %s.", r.Task, status)
	}
}
