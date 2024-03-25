package rand

import (
	"context"
	"time"
)

// Naive entropy seed. Uses time.Sleep jitter values.
//
// This RNG passes many (but not all) tests from dieharder suite, which is
// impressive considering that author is a complete amateur in the field.
// Use your own best judgement to decide whether this is good enough for your
// use case.
//
// Run dieharder test suite with a 5GB input file (takes ~12 hours):
//
//	$ make dieharder
func Seed(buf []byte) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nanos := make(chan int64)
	for i := 0; i < 48; i++ {
		go nanoGenerator(ctx, nanos)
	}
	const (
		mask         = 0b01001001
		maskShift    = 1
		masksPerByte = 3
	)
	for cursor := 0; cursor < len(buf); cursor++ {
		buf[cursor] = 0
		for chunk := 0; chunk < masksPerByte; {
			nano, ok := <-nanos
			if !ok {
				panic("jitter generator channel is closed: this is a bug")
			}

			// Drop trailing decimal zeroes (in case our timer is not granular enough)
			for nano%10 == 0 {
				nano /= 10
			}
			nano /= 10 // last decimal digit is not entirely random (it's never zero), so we drop it

			// Check that we have enough meaningful bits left
			if nano < 0xff {
				continue
			}

			buf[cursor] <<= maskShift
			buf[cursor] |= byte(nano) & mask
			chunk++
		}
	}
}

func nanoGenerator(ctx context.Context, results chan<- int64) {
	var start time.Time
	var delta, delay, jitter time.Duration
	const step time.Duration = 10000
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
			select {
			case results <- jitter.Nanoseconds():
			case <-ctx.Done():
				return
			}
			delay = delta
			if delay > step*13 {
				delay = step
			}
		}
	}
}
