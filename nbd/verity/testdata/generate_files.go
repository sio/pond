package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
)

func main() {
	dir := os.Args[1]
	count, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic(err)
	}
	size, err := strconv.Atoi(os.Args[3])
	if err != nil {
		panic(err)
	}
	random = rand.New(rand.NewSource(0x4815162342))
	for i := 0; i < count; i++ {
		writeRandom(filepath.Join(dir, fmt.Sprintf("%04d.file", i)), size)
	}
}

var random *rand.Rand

func writeRandom(path string, size int) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}
	defer func() { _ = file.Close() }()
	reader := &io.LimitedReader{
		R: random,
		N: int64(size),
	}
	_, err = io.Copy(file, reader)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s (%d bytes)\n", path, size)
}
