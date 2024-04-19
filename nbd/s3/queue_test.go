package s3

import (
	"testing"

	"context"
	"sync"
	"sync/atomic"
	"time"
)

func TestQueue(t *testing.T) {
	const (
		size   = 10
		rounds = 100
		delay  = time.Second / 1000
	)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	queue := NewQueue(ctx, size)
	t.Cleanup(func() { _ = queue.Close() })

	var normal, low atomic.Uint32
	var wg sync.WaitGroup
	for i := 0; i < size*rounds; i++ {
		wg.Add(1)
		if i%2 == 0 {
			go func() {
				queue.Acquire()
				normal.Add(1)
				wg.Done()
			}()
		} else {
			go func() {
				queue.AcquireLowPriority()
				low.Add(1)
				wg.Done()
			}()
		}
	}
	for i := 0; i < size*rounds/2; i++ {
		queue.Release()
	}
	var cur, prev, done uint32
	for {
		cur = normal.Load()
		if cur == prev {
			done++
			if done > 2 {
				break
			}
		}
		prev = cur
		time.Sleep(delay)
	}
	want := uint32(size * rounds / 2)
	got := normal.Load()
	if got != want {
		t.Errorf("unexpected normal priority result: want %d, got %d", want, got)
	}
	for i := 0; i < size*rounds/2; i++ {
		queue.Release()
	}
	wg.Wait()
	want = uint32(size * rounds / 2)
	got = low.Load()
	if got != want {
		t.Errorf("unexpected low priority result: want %d, got %d", want, got)
	}
}
