package journal

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Securely store information about sequential events in a structured
// (but schema-less) fashion
type Journal struct {
	// Input/output interface
	stream io.ReadWriter

	// Private key interface
	signer ssh.Signer

	// Header metadata
	version string
	ctime   time.Time

	// Current iteration state
	state []byte

	// List of cleanup functions
	cleanup []func()

	// Ensure thread safety of i/o operations
	write sync.Mutex

	// Never reuse closed Journal
	closed bool

	// Plain text message separator
	separator []byte
}

// Open file based journal
//
// Exclusive filesystem lock is automatically acquired.
// File will be created if not exists.
func Open(filename string, signer ssh.Signer) (*Journal, error) {
	lock := new(lockfile)
	err := lock.TryLock(filename + ".lock")
	if err != nil {
		return nil, err
	}
	fail := func(format string, a ...any) (*Journal, error) {
		lock.Unlock()
		return nil, fmt.Errorf(format, a...)
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fail("open: %w", err)
	}
	jrn, err := New(file, signer)
	if err != nil {
		_ = file.Close()
		return fail("%w", err)
	}
	jrn.Defer(func() { _ = file.Close() })
	jrn.Defer(lock.Unlock)
	return jrn, nil
}

// Use provided data stream for journal
//
// If provided stream implements io.Seeker interface journal will automatically
// jump to the start of the stream. Otherwise it is assumed that current
// position is already at the start of the stream.
//
// Exclusive access to underlying writer is assumed. Caller must handle locking
// on their own. For simple file-based logs use Open function which handles
// locking automatically.
func New(rw io.ReadWriter, s ssh.Signer) (*Journal, error) {
	if seeker, ok := rw.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("seek(0): %w", err)
		}
	}
	jrn := &Journal{
		stream: rw,
		signer: s,
	}
	err := jrn.parseHeader()
	if errors.Is(err, errEmptyStream) {
		err = jrn.writeHeader()
	}
	if err != nil {
		return nil, err
	}
	return jrn, nil
}

// Schedule a function to be called upon Journal.Close()
func (j *Journal) Defer(f func()) {
	j.cleanup = append(j.cleanup, f)
}

// Execute all scheduled cleanup functions in reverse chronological order
func (j *Journal) Close() {
	for i := len(j.cleanup) - 1; i >= 0; i-- {
		j.cleanup[i]()
	}
	j.closed = true
}

// Append a message to journal
func (j *Journal) Append(m *Message) error {
	if j.closed {
		return errors.New("writing to a closed journal")
	}
	j.write.Lock()
	defer j.write.Unlock()
	items, err := json.Marshal(m.Items)
	if err != nil {
		return fmt.Errorf("json: %w", err)
	}
	timeOffset := m.Timestamp.Sub(j.ctime).Seconds()
	var plaintext []byte
	plaintext = binary.BigEndian.AppendUint32(nil, uint32(timeOffset))
	plaintext = append(plaintext, byte(m.Action))
	plaintext = append(plaintext, items...)
	cipher, err := j.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("journal encrypt: %w", err)
	}
	_, err = j.stream.Write(append(cipher, j.separator...))
	if err != nil {
		return fmt.Errorf("journal write: %w", err)
	}
	return nil
}

// Construct a message and append it to journal
func (j *Journal) Message(action Verb, keyvals ...string) error {
	if len(keyvals)%2 != 0 {
		return errors.New("each key must be followed by a matching value")
	}
	var items = make([]Item, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		items[i/2] = Item{
			Name:  keyvals[i],
			Value: keyvals[i+1],
		}
	}
	return j.Append(&Message{
		Action:    action,
		Items:     items,
		Timestamp: time.Now(),
	})
}

func (j *Journal) ready() bool {
	return j.stream != nil && j.signer != nil
}

func (j *Journal) CatchUp() { // TODO
}

func (j *Journal) Next() *Message { // TODO
	return nil
}
