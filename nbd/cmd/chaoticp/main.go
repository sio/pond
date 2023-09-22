// Chaotic file copier
//
// Simulates random file access by copying data with multiple threads

package main

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"sync"
	"time"
)

const (
	chunkSize   = 16 * 1024
	numWorkers  = 10
	enableDebug = false
)

type chunk int32

func main() {
	if len(os.Args) != 3 {
		fatal("Exactly two arguments are required:\n\t%s <src> <dest>", os.Args[0])
	}
	if _, err := os.Stat(os.Args[2]); !errors.Is(err, fs.ErrNotExist) {
		fatal("Destination file exists, not overwriting: %s", os.Args[2])
	}

	src, err := os.Open(os.Args[1])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = src.Close() }()

	dest, err := os.OpenFile(os.Args[2], os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = dest.Close() }()

	stat, err := src.Stat()
	if err != nil {
		fatal(err)
	}
	size := stat.Size()
	err = dest.Truncate(size)
	if err != nil {
		fatal(err)
	}
	totalChunks := size / chunkSize
	chunks := make([]chunk, totalChunks)
	for i := 0; i < len(chunks); i++ {
		chunks[i] = chunk(i)
	}
	rand.Shuffle(len(chunks), func(i, j int) {
		chunks[i], chunks[j] = chunks[j], chunks[i]
	})
	debug(chunks)

	queue := make(chan chunk)
	go func() {
		for _, c := range chunks {
			queue <- c
		}
		close(queue)
	}()

	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		copier := chunkCopier{
			id:        i,
			src:       src,
			dest:      dest,
			totalSize: size,
			chunkSize: chunkSize,
		}
		go func() {
			defer wg.Done()
			for {
				c, ok := <-queue
				if !ok {
					break
				}
				copier.Copy(c)
			}
		}()
	}
	wg.Wait()

	// Sync is needed only for correct time/speed measurement,
	// reads will work correctly even before file is fully synced to disk
	err = dest.Sync()
	if err != nil {
		fatal(err)
	}

	elapsed := time.Since(start)
	speed := int64(float64(size) / elapsed.Seconds())
	info("Time %s", elapsed)
	info("Total %s (%d bytes)", humanBytes(size), size)
	info("Speed %s/s (%d bytes per second)", humanBytes(speed), speed)

	err = check(src, dest)
	if err != nil {
		fatal("Hash check failed: %v", err)
	}
	info("SHA512 OK")
}

func check(a, b *os.File) error {
	_, err := a.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = b.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	var aHash = sha512.New()
	_, err = io.Copy(aHash, a)
	if err != nil {
		return err
	}
	var bHash = sha512.New()
	_, err = io.Copy(bHash, b)
	if err != nil {
		return err
	}
	if !bytes.Equal(aHash.Sum(nil), bHash.Sum(nil)) {
		return fmt.Errorf("hash mismatch:\n%x\n%x", aHash.Sum(nil), bHash.Sum(nil))
	}
	return nil
}

var units = [...]string{"B", "KB", "MB", "GB", "TB"}

func humanBytes(b int64) string {
	const step = 1024
	var u int
	for u = 0; u < len(units); u++ {
		if b < step {
			break
		}
		b /= step
	}
	return fmt.Sprintf("%d%s", b, units[u])
}

type chunkCopier struct {
	id                   int
	src, dest            *os.File
	totalSize, chunkSize int64
	buf                  []byte
}

func (cc *chunkCopier) Copy(c chunk) {
	debug("Copier %d: chunk %d", cc.id, c)
	if cc.buf == nil {
		cc.buf = make([]byte, cc.chunkSize)
	}
	start := int64(c) * cc.chunkSize
	n, err := cc.src.ReadAt(cc.buf, start)
	if err != nil && !errors.Is(err, io.EOF) {
		fatal("Copier %d: reading chunk %d: %v", cc.id, c, err)
	}
	_, err = cc.dest.WriteAt(cc.buf[:n], start)
	if err != nil {
		fatal("Copier %d: writing chunk %d: %v", cc.id, c, err)
	}
}

func fatal(v any, a ...any) {
	out(os.Stderr, v, a...)
	os.Exit(1)
}

func info(v any, a ...any) {
	out(os.Stdout, v, a...)
}

func debug(v any, a ...any) {
	if !enableDebug {
		return
	}
	out(os.Stdout, v, a...)
}

func out(w io.Writer, v any, a ...any) {
	if len(a) == 0 {
		_, _ = fmt.Fprintln(w, v)
	} else {
		switch first := v.(type) {
		case string:
			if first[len(first)-1] != '\n' {
				first += "\n"
			}
			_, _ = fmt.Fprintf(w, first, a...)
		default:
			_, _ = fmt.Fprintln(w, append([]any{v}, a...)...)
		}
	}
}
