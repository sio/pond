package test

import (
	"testing"

	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/sio/pond/nbd/buffer"
	"github.com/sio/pond/nbd/s3"
)

func TestWithMinio(t *testing.T) {
	if testing.Short() {
		t.Skipf("this test takes quite some time")
	}
	directory := randomDir(t)
	server, access, secret := serve(t, directory)
	cacheDir, err := os.MkdirTemp("", "pond-cache-*")
	if err != nil {
		t.Fatalf("create cache directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(cacheDir) })
	for _, name := range []string{"10MB", "10KB"} {
		t.Run(name, func(t *testing.T) {
			cache, err := s3.Open(server, access, secret, "garbage", name, cacheDir)
			if err != nil {
				t.Fatalf("s3.Open: %v", err)
			}
			t.Cleanup(func() {
				err := cache.Close()
				if err != nil {
					t.Fatalf("close cache: %v", err)
				}
			})
			original, err := os.Open(filepath.Join(directory, "garbage", name))
			if err != nil {
				t.Fatalf("open original file: %v", err)
			}
			t.Cleanup(func() { _ = original.Close() })
			stat, err := original.Stat()
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			interim, err := os.Open(filepath.Join(cacheDir, name))
			if err != nil {
				t.Fatalf("open interim file: %v", err)
			}
			t.Cleanup(func() { _ = interim.Close() })

			for _, tt := range []struct {
				offset, size int64
			}{
				{10, 60},
				{0, 10 << 10},
				{1<<20 + 100, 1000},
				{8 << 20, 2 << 20},
				{1 << 20, 9 << 20},
				{0, 10 << 20},
			} {
				if tt.offset+tt.size > stat.Size() {
					continue
				}
				t.Run(fmt.Sprintf("@%s+%s", filesize(tt.offset), filesize(tt.size)), func(t *testing.T) {
					var sample = new(hashCollection)
					sample.cache, sample.ce = hash(cache, tt.offset, tt.size)
					sample.interim, sample.ie = hash(interim, tt.offset, tt.size)
					sample.original, sample.oe = hash(original, tt.offset, tt.size)

					err = sample.Validate()
					if err != nil {
						t.Error(err)
					}
					t.Log(sample)
				})
			}
		})
	}
}

type hashCollection struct {
	original, interim, cache []byte
	oe, ie, ce               error
}

func (hc *hashCollection) Validate() error {
	if err := errors.Join(hc.oe, hc.ie, hc.ce); err != nil {
		return err
	}
	var msg = new(strings.Builder)
	if !bytes.Equal(hc.original, hc.interim) {
		_, _ = fmt.Fprintf(msg, "original != interim (%x != %x)", hc.original, hc.interim)
	}
	if !bytes.Equal(hc.interim, hc.cache) {
		if msg.Len() > 0 {
			_, _ = msg.WriteString("; ")
		}
		_, _ = fmt.Fprintf(msg, "interim != cache (%x != %x)", hc.interim, hc.cache)
	}
	if msg.Len() == 0 {
		return nil
	}
	return errors.New(msg.String())
}

func (hc *hashCollection) String() string {
	return fmt.Sprintf("%x", hc.original)
}

func hash(r io.ReaderAt, offset, size int64) ([]byte, error) {
	buf := buffer.Get()
	defer buffer.Put(buf)

	h := sha256.New()
	for size > 0 {
		buf = buf[:min(cap(buf), int(size))]
		n, err := r.ReadAt(buf, offset)
		offset += int64(n)
		size -= int64(n)
		if err != nil {
			return nil, err
		}
		_, err = h.Write(buf)
		if err != nil {
			return nil, err
		}
		if size < 0 {
			panic("hash: we have read too much")
		}
	}
	return h.Sum(nil), nil
}

func serve(t *testing.T, directory string) (endpoint, access, secret string) {
	if testing.Short() || os.Getenv("TEST_SKIP_CONTAINERS") != "" {
		t.Skipf("skip long test requiring helper containers")
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	hostname := randomString()
	access = randomString()
	secret = randomString()

	minio, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			// Last Minio version that features `minio gateway nas`,
			// see https://github.com/minio/minio/issues/14331
			Image: "quay.io/minio/minio:RELEASE.2022-05-26T05-48-41Z",
			Name:  hostname,
			Cmd:   []string{"gateway", "nas", "/data"},
			Files: []testcontainers.ContainerFile{
				{
					HostFilePath:      directory,
					ContainerFilePath: "/",
					FileMode:          0o700,
				},
			},
			Env: map[string]string{
				"MINIO_ROOT_USER":     access,
				"MINIO_ROOT_PASSWORD": secret,
			},
			WaitingFor: wait.ForLog("You are running an older version of MinIO"),
		},
	})
	if err != nil {
		t.Fatalf("starting container: %v", err)
	}
	t.Cleanup(func() {
		if err := minio.Terminate(ctx); err != nil {
			t.Fatalf("terminating container: %v", err)
		}
	})
	host, err := minio.Host(ctx)
	if err != nil {
		t.Fatalf("get hostname: %v", err)
	}
	port, err := minio.MappedPort(ctx, "9000/tcp")
	if err != nil {
		t.Fatalf("get port: %v", err)
	}
	endpoint = fmt.Sprintf("%s:%s", host, port.Port())
	t.Log("Started an S3 server for tests: ")
	t.Logf("$ mc alias set testserver http://%s %q %q", endpoint, access, secret)
	return endpoint, access, secret
}

func randomDir(t *testing.T) string {
	directory, err := os.MkdirTemp("", "pond-*")
	if err != nil {
		t.Fatalf("create temp directory: %v", err)
	}
	t.Cleanup(func() {
		err = os.RemoveAll(directory)
		if err != nil {
			t.Fatalf("remove %s: %v", directory, err)
		}
	})

	datadir := filepath.Join(directory, "data")
	for _, size := range []int64{10 << 10, 10 << 20} {
		err := randomFile(
			filepath.Join(datadir, "garbage", filesize(size)),
			size,
		)
		if err != nil {
			t.Fatalf("create file with random data: %v", err)
		}
	}
	return datadir
}

func randomFile(path string, size int64) error {
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	err = file.Truncate(0)
	if err != nil {
		return fmt.Errorf("truncate %s: %w", path, err)
	}
	_, err = io.CopyN(file, rand.Reader, size)
	if err != nil {
		return fmt.Errorf("copying random data to %s: %w", path, err)
	}
	return nil
}

func filesize(size int64) string {
	const ceiling = 1 << 10
	var unit = []string{"B", "KB", "MB", "GB", "TB", "PB"}
	var suffix int
	for size >= ceiling && suffix+1 < len(unit) {
		size >>= 10
		suffix++
	}
	return fmt.Sprintf("%d%s", size, unit[suffix])
}

func randomString() string {
	var buf [10]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		panic("random: " + err.Error())
	}
	return fmt.Sprintf("%x", buf)
}
