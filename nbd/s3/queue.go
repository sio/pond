package s3

import (
	"context"
)

func NewQueue(ctx context.Context, size int) *Queue {
	q := &Queue{
		global: make(chan struct{}, size),
		normal: make(chan struct{}, size),
		low:    make(chan struct{}, size),
	}
	q.ctx, q.cancel = context.WithCancel(ctx)
	return q
}

// Simple two-level priority queue
//
// Until queue is in active use (global semaphore full) there is little
// difference between low priority and normal Acquire. When there is a
// significant resource contention, Acquire() will be prioritized over
// AcquireLowPriority()
type Queue struct {
	global, normal, low chan struct{}
	ctx                 context.Context
	cancel              context.CancelFunc
}

func (q *Queue) Close() error {
	if q.cancel != nil {
		q.cancel()
	}
	return nil
}

// Calling Release() without previously calling Acquire() or
// AcquireLowPriority() will result in dead lock.
func (q *Queue) Release() {
	<-q.global
	select {
	case <-q.normal:
		return
	default:
	}
	select {
	case <-q.normal:
	case <-q.low:
	}
}

func (q *Queue) Acquire() {
	q.normal <- struct{}{}
	q.global <- struct{}{}
}

func (q *Queue) AcquireLowPriority() {
	q.low <- struct{}{}
	q.global <- struct{}{}
}
