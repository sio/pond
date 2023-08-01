package audit

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Securely store information about sequential events in a structured
// (but schema-less) fashion
type Log struct {
	// Input/output interface
	stream io.ReadWriter

	// Creation time of log file.
	// May or may not match the timestamp of the first message
	ctime time.Time

	// List of cleanup functions
	cleanup []func()

	// Ensure thread safety of i/o operations
	write sync.Mutex

	// Never reuse closed Log
	closed bool

	// Plain text message separator
	separator []byte

	// Current iteration state
	state []byte
}

// Open audit log based on a file
//
// Exclusive filesystem lock is automatically acquired.
// File will be created if not exists.
func Open(filename string) (*Log, error) {
	lock := new(lockfile)
	err := lock.TryLock(filename + ".lock")
	if err != nil {
		return nil, err
	}
	fail := func(format string, a ...any) (*Log, error) {
		lock.Unlock()
		return nil, fmt.Errorf(format, a...)
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fail("open: %w", err)
	}
	log, err := New(file)
	if err != nil {
		_ = file.Close()
		return fail("%w", err)
	}
	log.Defer(func() { _ = file.Close() })
	log.Defer(lock.Unlock)
	return log, nil
}

// Use provided data stream for audit log
//
// If provided stream implements io.Seeker interface log will automatically
// jump to the start of the stream. Otherwise it is assumed that current
// position is already at the start of the stream.
//
// Exclusive access to underlying writer is assumed. Caller must handle locking
// on their own. For simple file-based logs use Open function which handles
// locking automatically.
func New(rw io.ReadWriter) (*Log, error) {
	if seeker, ok := rw.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("seek(0): %w", err)
		}
	}
	log := &Log{stream: rw}
	return log, nil
}

// Schedule a function to be called upon Log.Close()
func (a *Log) Defer(f func()) {
	a.cleanup = append(a.cleanup, f)
}

// Execute all scheduled cleanup functions in reverse chronological order
func (a *Log) Close() {
	for i := len(a.cleanup) - 1; i >= 0; i-- {
		a.cleanup[i]()
	}
	a.closed = true
}

// Append a message to audit log
func (a *Log) Append(m *Message) error {
	if a.closed {
		return errors.New("writing to a closed log")
	}
	a.write.Lock()
	defer a.write.Unlock()
	items, err := json.Marshal(m.Items)
	if err != nil {
		return fmt.Errorf("json: %w", err)
	}
	timeOffset := m.Timestamp.Sub(a.ctime).Seconds()
	var plaintext []byte
	plaintext = binary.BigEndian.AppendUint32(nil, uint32(timeOffset))
	plaintext = append(plaintext, byte(m.Action))
	plaintext = append(plaintext, items...)
	cipher, err := a.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("log encrypt: %w", err)
	}
	_, err = a.stream.Write(append(cipher, a.separator...))
	if err != nil {
		return fmt.Errorf("log write: %w", err)
	}
	return nil
}

// Construct a message and append it to audit log
func (a *Log) Message(action Verb, keyvals ...string) error {
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
	return a.Append(&Message{
		Action:    action,
		Items:     items,
		Timestamp: time.Now(),
	})
}

func (a *Log) CatchUp() { // TODO
}

func (a *Log) Next() *Message { // TODO
	return nil
}
