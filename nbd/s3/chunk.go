package s3

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sio/pond/nbd/logger"
)

const (
	// Target network speed is 100Mbps.
	//
	// TLS handshake takes around 500ms (at least 3 network roundtrips plus crypto),
	// that translates into 100/8*0.500 = 6.25MB transfer (lost opportunity).
	// We need to amortize this overhead.
	//
	// Amazon recommends to use 8..16MB for S3 range requests:
	// https://docs.aws.amazon.com/whitepapers/latest/s3-optimizing-performance-best-practices/use-byte-range-fetches.html
	//
	// TODO: does minio.Client maintain a connection pool for HTTP requests?
	chunkSize    = 8 << 20
	chunkVersion = "ChunkMapV02" // always change this when chunkSize is changed
)

type chunk uint64

type chunkMap struct {
	path string
	size uint64

	bitmap    *big.Int
	bitmapMu  sync.RWMutex
	running   map[chunk]chan struct{}
	runningMu sync.Mutex
	modified  time.Time
	saved     time.Time
}

func (m *chunkMap) Offset(c chunk) (offset, size int64) {
	size = int64(chunkSize)
	offset = size * int64(c)
	total := int64(m.size)
	if offset > total {
		return 0, 0
	}
	if offset+size > total {
		size = total - offset
	}
	return offset, size
}

func openChunkMap(path string, size int64) (*chunkMap, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}

	c := &chunkMap{
		path:    path,
		size:    uint64(size),
		bitmap:  new(big.Int),
		running: make(map[chunk]chan struct{}),
	}

	var header chunkMapExportHeader
	err = binary.Read(file, binary.BigEndian, &header)
	if err != nil {
		if stat.Size() == 0 {
			return c, nil
		}
		return nil, fmt.Errorf("chunk map header: %w", err)
	}
	version := string(header.Version[:])
	if len(version) < 9 || version[:9] != chunkVersion[:9] {
		return nil, fmt.Errorf("invalid chunk map version: %s", version)
	}
	if version != chunkVersion {
		// If we receive chunk map with incompatible header, act as if we have
		// no cached data at all
		return c, nil // TODO: add backward compaitibility with previous chunkMap formats
	}
	if header.ChunkSize != chunkSize || header.TotalSize != c.size {
		// Drop cache on any irregularities
		return c, nil
	}
	const safeChunkByteCeiling = 10 << 20 // (10<<20) of (1<<20) chunks == 10TB of data, we'll never encounter that much
	chunkByteCount := stat.Size() - int64(binary.Size(header))
	if chunkByteCount > int64(size/chunkSize) || chunkByteCount > safeChunkByteCeiling {
		return nil, fmt.Errorf("chunk map too large: %dMB (%d bytes)", stat.Size()<<20, stat.Size())
	}
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading chunk map: %w", err)
	}
	c.bitmap.SetBytes(raw)
	c.saved = time.Now()
	return c, nil
}

func (m *chunkMap) Close() error {
	return m.Save()
}

// Mark chunk as done
func (m *chunkMap) Done(c chunk) {
	m.bitmapMu.Lock()
	defer m.bitmapMu.Unlock()
	m.bitmap.SetBit(m.bitmap, int(c), 1)

	m.runningMu.Lock()
	ch, ok := m.running[c]
	if ok {
		close(ch)
		delete(m.running, c)
	}
	m.runningMu.Unlock()

	m.modified = time.Now()
}

// Check if chunk is already done
func (m *chunkMap) Check(c chunk) (wait <-chan struct{}, done bool) {
	return m.check(c)
}

func (m *chunkMap) check(c chunk) (ch chan struct{}, done bool) {
	m.bitmapMu.RLock()
	defer m.bitmapMu.RUnlock()

	done = m.bitmap.Bit(int(c)) == 1
	if done {
		return closed, true
	}

	m.runningMu.Lock()
	defer m.runningMu.Unlock()

	ch, ok := m.running[c]
	if !ok {
		ch = make(chan struct{})
		m.running[c] = ch
	}

	return ch, false
}

// Save chunkMap to file system for persistence
func (m *chunkMap) Save() error {
	temp, err := os.CreateTemp(filepath.Dir(m.path), filepath.Base(m.path)+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temp.Name()) }()

	m.bitmapMu.RLock()
	defer m.bitmapMu.RUnlock()

	header := chunkMapExportHeader{
		ChunkSize: uint64(chunkSize),
		TotalSize: m.size,
	}
	n := copy(header.Version[:], []byte(chunkVersion))
	if n < len(chunkVersion) {
		panic("chunk version is longer than supported by on-disk format. This is a bug!")
	}
	err = binary.Write(temp, binary.BigEndian, header)
	if err != nil {
		return fmt.Errorf("writing header: %w", err)
	}
	_, err = temp.Write(m.bitmap.Bytes())
	if err != nil {
		return fmt.Errorf("writing bitmap: %w", err)
	}
	err = os.Rename(temp.Name(), m.path)
	if err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	m.saved = time.Now()
	return nil
}

func (m *chunkMap) AutoSave(ctx context.Context) {
	const autoSaveInterval = 9 * time.Minute
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(autoSaveInterval / 2):
		}
		if m.saved.After(m.modified) {
			continue
		}
		if m.modified.Add(autoSaveInterval/4).After(time.Now()) && m.modified.Sub(m.saved) < autoSaveInterval {
			continue
		}
		err := m.Save()
		if err != nil {
			log := logger.FromContext(ctx)
			log.Warn("chunkmap autosave failed", "error", err)
		}
	}
}

type chunkMapExportHeader struct {
	Version   [16]byte
	ChunkSize uint64
	TotalSize uint64
}

var closed = func() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()
