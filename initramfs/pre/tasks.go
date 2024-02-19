package pre

import (
	"fmt"
	"strings"
	"sync"
)

type TaskManager struct {
	tasks map[string]chan struct{}
	lock  sync.Mutex
}

func (m *TaskManager) Status() string {
	m.lock.Lock()
	defer m.lock.Unlock()
	waiting := make([]string, 0, len(m.tasks))
	complete := 0
	for task, ch := range m.tasks {
		select {
		case <-ch:
			complete++
		default:
			waiting = append(waiting, task)
		}
	}
	if len(waiting) > 0 {
		return fmt.Sprintf("Blocked. %d tasks complete. Waiting for %s", complete, strings.Join(waiting, ", "))
	} else {
		return fmt.Sprintf("Not blocked. %d tasks complete.", complete)
	}
}

func (m *TaskManager) Wait(tasks ...string) {
	if len(tasks) == 0 {
		m.lock.Lock()
		defer m.lock.Unlock()
		for _, ch := range m.tasks {
			<-ch
		}
		return
	}
	for _, task := range tasks {
		ch := m.ch(task)
		<-ch
	}
}

func (m *TaskManager) ch(task string) chan struct{} {
	ch, ok := m.tasks[task]
	if !ok {
		m.lock.Lock()
		defer m.lock.Unlock()
		ch = make(chan struct{})
		if m.tasks == nil {
			m.tasks = make(map[string]chan struct{})
		}
		m.tasks[task] = ch
	}
	return ch
}

func (m *TaskManager) Done(task string) {
	ch := m.ch(task)
	m.lock.Lock()
	defer m.lock.Unlock()
	close(ch)
}
