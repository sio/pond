// Simple stress tests for arbitrary secretd server
package stress

// TODO: add benchmark against synthetic server running with a custom multicore ssh-agent

import (
	"testing"

	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

func BenchmarkServerReply(b *testing.B) {
	addr := os.Getenv("SECRETD_BENCH_SERVER")
	if addr == "" {
		b.Skip("server not specified: $SECRETD_BENCH_SERVER")
	}
	server, err := url.Parse(addr)
	if err != nil {
		b.Fatalf("could not parse $SECRETD_BENCH_SERVER url: %v", err)
	}
	query := os.Getenv("SECRETD_BENCH_QUERY")
	if query == "" {
		b.Fatal("benchmark query not specified: $SECRETD_BENCH_QUERY (comma separated secret names)")
	}
	queryItems := strings.Split(query, ",")
	query = fmt.Sprintf(`["%s"]`, strings.Join(queryItems, `","`))
	keyPath := os.Getenv("SECRETD_BENCH_CLIENT_KEY")
	if keyPath == "" {
		b.Fatal("client key not specified: $SECRETD_BENCH_CLIENT_KEY")
	}
	keyRaw, err := os.ReadFile(keyPath)
	if err != nil {
		b.Fatalf("reading client key: %v", err)
	}
	key, err := ssh.ParsePrivateKey(keyRaw)
	if err != nil {
		b.Fatalf("parsing client key: %v", err)
	}
	clientConf := &ssh.ClientConfig{
		User: "n/a",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	iterate := func(stdin, stdout *bytes.Buffer) {
		stdin.Reset()
		stdout.Reset()
		client, err := ssh.Dial(server.Scheme, server.Host, clientConf)
		if err != nil {
			b.Fatalf("ssh dial: %v", err)
		}
		defer func() { _ = client.Close() }()
		session, err := client.NewSession()
		if err != nil {
			b.Fatalf("ssh session: %v", err)
		}
		defer func() { _ = session.Close() }()
		_, err = fmt.Fprintln(stdin, query)
		if err != nil {
			b.Fatalf("writing to stdin buffer: %v", err)
		}
		session.Stdin = stdin
		session.Stdout = stdout
		err = session.Shell()
		if err != nil {
			b.Fatalf("session shell: %v", err)
		}
		err = session.Wait()
		if err != nil {
			b.Fatalf("session error: %v", err)
		}
		var r = new(reply)
		err = json.Unmarshal(stdout.Bytes(), r)
		if err != nil {
			b.Fatalf("json reply: %v", err)
		}
		if len(r.Errors) != 0 || len(r.Secrets) != len(queryItems) {
			b.Fatalf("unexpected reply: %s", stdout.String())
		}
		for _, item := range queryItems {
			if len(r.Secrets[item]) == 0 {
				b.Fatalf("missing value for secret %q: %s", item, stdout.String())
			}
		}
	}

	// Saturate all available CPUs with stress testing
	//
	// Multiple requests per CPU are used to account for i/o pauses.
	// Specific number (5 workers per CPU) was chosen empirically after
	// a round of manual tests.
	const workersPerCPU = 5
	var wg sync.WaitGroup
	next := make(chan bool)
	for i := 0; i < runtime.NumCPU()*workersPerCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stdin := new(bytes.Buffer)
			stdout := new(bytes.Buffer)
			for {
				iterate(stdin, stdout)
				_, ok := <-next // accept next job only after finishing the previous one
				if !ok {
					return
				}
			}
		}()
	}
	for i := 0; i < b.N; i++ {
		select {
		case next <- true:
			// block until one of previous jobs is finished
		case <-time.After(5 * time.Second):
			b.Fatal("benchmark likely deadlocked")
		}
	}
	close(next)
	wg.Wait()
}

type reply struct {
	Secrets map[string]string
	Errors  []string
}
