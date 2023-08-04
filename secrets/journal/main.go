package journal

import (
	"bufio"
	"bytes"
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
	// Data
	stream io.ReadWriter
	signer ssh.Signer

	// Metadata
	version string
	ctime   time.Time

	// State
	state     []byte
	separator []byte
	scanner   *bufio.Scanner
	lock      sync.Mutex
	closed    bool
	cleanup   []func()
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
	if j.closed {
		return
	}
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
	j.lock.Lock()
	defer j.lock.Unlock()
	var err error
	if err = j.catchup(); err != nil {
		return err
	}
	plaintext, err := m.Encode(j.ctime)
	if err != nil {
		return fmt.Errorf("message serialization: %w", err)
	}
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
func (j *Journal) Message(action Verb, elements ...string) error {
	return j.Append(&Message{
		Action:    action,
		Elements:  elements,
		Timestamp: time.Now(),
	})
}

func (j *Journal) ready() bool {
	return j.stream != nil && j.signer != nil
}

// Read all messages until the end of journal without returning their contents
func (j *Journal) catchup() error {
	var err error
	var count uint
	for err == nil {
		_, err = j.fetchNext()
		count++
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	return fmt.Errorf("catching up with journal, entry %d: %w", count, err)
}

// Read the next message in journal. Returns io.EOF after the last message.
func (j *Journal) Next() (*Message, error) {
	j.lock.Lock()
	defer j.lock.Unlock()
	return j.fetchNext()
}

// Fetch next message without locking the journal. Use with caution!
func (j *Journal) fetchNext() (*Message, error) {
	if !j.ready() {
		return nil, fmt.Errorf("can not read from uninitialized journal")
	}
	if len(j.state) == 0 || len(j.separator) == 0 {
		return nil, fmt.Errorf("can not read messages before parsing header")
	}
	if j.scanner == nil {
		j.scanner = bufio.NewScanner(j.stream)
		j.scanner.Split(j.splitFunc)
	}
	if !j.scanner.Scan() {
		err := j.scanner.Err()
		if err == nil {
			err = io.EOF
		}
		return nil, err
	}
	cipher := j.scanner.Bytes()
	if len(cipher) == 0 {
		return j.fetchNext()
	}
	plain, err := j.decrypt(cipher)
	if err != nil {
		return nil, err
	}
	var message Message
	err = message.Decode(plain, j.ctime)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// bufio.SplitFunc for reading journal messages
func (j *Journal) splitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	token, _, found := bytes.Cut(data, j.separator)
	if len(token) == 0 {
		token = nil
	}
	if atEOF {
		err = bufio.ErrFinalToken
	}
	if found {
		return len(token) + len(j.separator), token, err
	}
	return len(token), token, err
}
