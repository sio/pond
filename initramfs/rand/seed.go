package rand

import (
	"context"
	"time"
)

// Naive entropy seed. Uses time.Sleep jitter values.
//
// Dieharder tests show that this RNG's quality is abysmall.
// Use only in non-critical deployments.
func Seed(buf []byte) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nanos := make(chan int64)
	for i := 0; i < 48; i++ {
		go nanoGenerator(ctx, nanos)
	}
	var i int
	for nano := range nanos {
		if i >= len(buf) {
			return
		}

		// Drop trailing zeroes (in case our timer is not granular enough)
		for nano&1 == 0 {
			nano = nano >> 1
		}

		// Drop the last bit because it's always 1, and the bit after that for no good reason
		nano = nano >> 2

		if nano < 0xff {
			continue // Possibly contains meaningless leading zero bits
		}

		buf[i] = byte(nano & 0xff)
		i++
	}
}

func nanoGenerator(ctx context.Context, results chan<- int64) {
	var start time.Time
	var delta, delay, jitter time.Duration
	const step time.Duration = 11579 // nanoseconds = 11us; best case 1byte/11us = 84KB/s = 21k int32/s
	delay = step
	for {
		start = time.Now()
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			delta = time.Since(start)
			if delta == delay {
				continue
			}
			jitter = delta - delay
			if jitter < 0 {
				jitter *= -1
			}
			results <- jitter.Nanoseconds()
			delay = delta
			if delay > step*13 {
				delay = step
			}
		}
	}
}
