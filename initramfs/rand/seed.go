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
		mask         = 0b00001111
		maskBits     = 4
		masksPerByte = 2
	)
	for cursor := 0; cursor < len(buf); cursor++ {
		buf[cursor] = 0
		for chunk := 0; chunk < masksPerByte; {
			nano, ok := <-nanos
			if !ok {
				panic("jitter generator channel is closed: this is a bug")
			}

			// Drop trailing zeroes (in case our timer is not granular enough)
			for nano&1 == 0 {
				nano = nano >> 1
			}

			// Drop the last bit because it's always 1, and the bit after that for no good reason
			nano = nano >> (1 + chunk)

			if nano < 0xff {
				continue // Possibly contains meaningless leading zero bits
			}

			buf[cursor] <<= maskBits
			buf[cursor] |= byte(nano) & mask
			chunk++
		}
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
