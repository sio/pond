package s3

import (
	"context"
)

const (
	// Maximum number of simultaneous connections per S3 object
	connLimitPerObject = 16

	// Total maximum number of simultaneous connections across all S3 objects
	connLimitGlobal = 64
)

var globalConnectionQueue = NewQueue(context.Background(), connLimitGlobal)

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
func (q *Queue) Release() error {
	select {
	case <-q.global:
	case <-q.ctx.Done():
		return context.Cause(q.ctx)
	}
	select {
	case <-q.normal:
		return nil
	default:
	}
	select {
	case <-q.normal:
	case <-q.low:
	case <-q.ctx.Done():
		return context.Cause(q.ctx)
	}
	return nil
}

func (q *Queue) Acquire() error {
	select {
	case q.normal <- struct{}{}:
	case <-q.ctx.Done():
		return context.Cause(q.ctx)
	}
	select {
	case q.global <- struct{}{}:
	case <-q.ctx.Done():
		return context.Cause(q.ctx)
	}
	return nil
}

func (q *Queue) AcquireLowPriority() error {
	select {
	case q.low <- struct{}{}:
	case <-q.ctx.Done():
		return context.Cause(q.ctx)
	}
	select {
	case q.global <- struct{}{}:
	case <-q.ctx.Done():
		return context.Cause(q.ctx)
	}
	return nil
}
