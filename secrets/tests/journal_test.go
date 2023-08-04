package tests

import (
	"testing"

	"secrets/journal"

	"crypto/ed25519"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"path/filepath"
)

func BenchmarkJournalWrite(b *testing.B) {
	j, err := setupJournal()
	if err != nil {
		b.Fatal(err)
	}
	defer j.Cleanup()

	const (
		maxLen = 10
		word   = "HelloWorldFooBar\r\n!!"
	)
	for i := 0; i < b.N; i++ {
		elements := make([]string, i%(maxLen-1)+1)
		for e := 0; e < len(elements); e++ {
			elements[e] = word
		}
		err := j.Message(journal.Add, elements...)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestJournalWrite(t *testing.T) {
	j, err := setupJournal()
	if err != nil {
		t.Fatal(err)
	}
	defer j.Cleanup()
	var messages = []string{
		"hello",
		"world",
		"FOO",
		"BARBAZ!",
		"\n",
		"\r\n",
		"\U00023123",
	}
	for i := 0; i < len(messages); i++ {
		elements := make([]string, i+1)
		for index := 0; index < len(elements); index++ {
			elements[index] = messages[i]
		}
		err = j.Message(journal.Add, elements...)
		if err != nil {
			t.Fatalf("failed to write to journal: word=%q, repeat=%d", messages[i], i+1)
		}
	}

	err = j.Reopen()
	if err != nil {
		t.Fatalf("failed to reopen journal: %v", err)
	}
	for i := 0; i < len(messages); i++ {
		elements := make([]string, i+1)
		for index := 0; index < len(elements); index++ {
			elements[index] = messages[i]
		}
		err = j.Message(journal.Add, elements...)
		if err != nil {
			t.Fatalf("failed to write to journal after reopening: word=%q, repeat=%d", messages[i], i+1)
		}
	}

	err = j.Reopen()
	if err != nil {
		t.Fatalf("failed to reopen journal the second time: %v", err)
	}
	for i := 0; i < 2*len(messages); i++ {
		m, err := j.Next()
		if err != nil {
			t.Fatalf("message %d: %v", i, err)
		}
		if m.Action != journal.Add {
			t.Errorf("message %d: incorrect action: %c", i, m.Action)
		}
		if len(m.Elements) != i%len(messages)+1 {
			t.Errorf("message %d: unexpected length %d instead of %d", i, len(m.Elements), i%len(messages))
		}
		for index, element := range m.Elements {
			if element != messages[i%len(messages)] {
				t.Errorf("message %d: unexpected element %d: got %q, want %q", i, index, element, messages[index])
			}
		}
		t.Logf("message %d: %q [OK]", i, m.Elements[0])
	}
}

type testJournal struct {
	*journal.Journal
	Cleanup func()
	path    string
	signer  ssh.Signer
}

func (j *testJournal) Reopen() error {
	j.Close()
	var err error
	j.Journal, err = journal.Open(j.path, j.signer)
	return err
}

func setupJournal() (j *testJournal, err error) {
	dir, err := os.MkdirTemp("", "pond-journal-test-*")
	if err != nil {
		return nil, err
	}
	j = new(testJournal)
	j.Cleanup = func() {
		j.Close()
		os.RemoveAll(dir)
	}
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("generating ssh key: %w", err)
	}
	j.signer, err = ssh.NewSignerFromKey(key)
	if err != nil {
		return nil, fmt.Errorf("opening key for signing: %w", err)
	}
	j.path = filepath.Join(dir, "journal")
	j.Journal, err = journal.Open(j.path, j.signer)
	if err != nil {
		return nil, fmt.Errorf("failed to open journal: %w", err)
	}
	return j, nil
}
