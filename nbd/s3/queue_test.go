package s3

import (
	"testing"

	"context"
	"math/rand"
	"strings"
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

	var progress strings.Builder
	t.Cleanup(func() { t.Logf("queue progress visualised:\n%s", progress.String()) })

	var normal, low atomic.Uint32
	var wg sync.WaitGroup
	for i := 0; i < size*rounds; i++ {
		wg.Add(1)
		if i%2 == 0 {
			go func() {
				defer wg.Done()
				err := queue.Acquire(ctx)
				if err != nil {
					t.Errorf("acquire: %v", err)
				}
				progress.WriteRune(':')
				normal.Add(1)
			}()
		} else {
			go func() {
				defer wg.Done()
				err := queue.AcquireLowPriority(ctx)
				if err != nil {
					t.Errorf("acquire: %v", err)
				}
				progress.WriteRune('.')
				low.Add(1)
			}()
		}
	}
	time.Sleep(delay)
	n, l := normal.Load(), low.Load()
	t.Logf("just started: low=%d, normal=%d", l, n)
	if l > size || n > size || (n+l) > size {
		t.Errorf("too many tasks got through before the first Release(): low=%d, normal=%d", l, n)
	}
	progress.WriteRune('\n')
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
	n, l = normal.Load(), low.Load()
	t.Logf("halfway there: low=%d, normal=%d", l, n)
	if l > n/2 { // TODO: this test is flakey in verbose mode, output to console skews timing just enough
		t.Errorf("too many low priority tasks succeeded: low=%d, normal=%d", l, n)
	}
	progress.WriteRune('\n')
	for i := 0; i < size*rounds/2; i++ {
		err := queue.Release()
		if err != nil {
			t.Fatalf("release: %v", err)
		}
	}
	wg.Wait()
	want := uint32(size * rounds / 2)
	n, l = normal.Load(), low.Load()
	if n != want || l != want {
		t.Errorf("unexpected final result: low=%d, normal=%d (want %d)", l, n, want)
	}
}

func BenchmarkQueue(b *testing.B) {
	const size = 16
	ctx := context.Background()
	queue := NewQueue(ctx, size)
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
			err = queue.AcquireLowPriority(ctx)
		} else {
			err = queue.Acquire(ctx)
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
