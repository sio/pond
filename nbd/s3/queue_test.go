package s3

import (
	"testing"

	"context"
	"math/rand"
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
				err := queue.Acquire()
				if err != nil {
					t.Errorf("acquire: %v", err)
				}
				normal.Add(1)
				wg.Done()
			}()
		} else {
			go func() {
				err := queue.AcquireLowPriority()
				if err != nil {
					t.Errorf("acquire: %v", err)
				}
				low.Add(1)
				wg.Done()
			}()
		}
	}
	for i := 0; i < size*rounds/2; i++ {
		err := queue.Release()
		if err != nil {
			t.Fatalf("release: %v", err)
		}
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
		err := queue.Release()
		if err != nil {
			t.Fatalf("release: %v", err)
		}
	}
	wg.Wait()
	want = uint32(size * rounds / 2)
	got = low.Load()
	if got != want {
		t.Errorf("unexpected low priority result: want %d, got %d", want, got)
	}
}

func BenchmarkQueue(b *testing.B) {
	const size = 16
	queue := NewQueue(context.Background(), size)
	b.Cleanup(func() { _ = queue.Close() })
	tick := make(chan struct{})
	defer close(tick)
	errs := make(chan error, 1)
	go func() {
		for {
			_, ok := <-tick
			if !ok {
				return
			}
			err := queue.Release()
			if err != nil {
				errs <- err
				return
			}
		}
	}()
	for i := 0; i < b.N; i++ {
		var err error
		if rand.Intn(3)%3 == 0 {
			err = queue.AcquireLowPriority()
		} else {
			err = queue.Acquire()
		}
		if err != nil {
			b.Fatalf("acquire: %v", err)
		}
		select {
		case tick <- struct{}{}:
		case err = <-errs:
			b.Fatalf("release: %v", err)
		}
	}
}
