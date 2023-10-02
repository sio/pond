package stress

import (
	"testing"

	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh"
	"net/url"
	"os"
)

const query = `["hello", "world"]`

func BenchmarkServerReply(b *testing.B) {
	addr := os.Getenv("SECRETD_BENCH_SERVER")
	if addr == "" {
		b.Skip("server not specified: $SECRETD_BENCH_SERVER")
	}
	server, err := url.Parse(addr)
	if err != nil {
		b.Fatalf("could not parse $SECRETD_BENCH_SERVER url: %v", err)
	}
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
	stdin := new(bytes.Buffer)
	stdout := new(bytes.Buffer)
	for i := 0; i < b.N; i++ {
		stdin.Reset()
		stdout.Reset()
		func() {
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
			err = json.NewDecoder(stdout).Decode(r)
			if err != nil {
				b.Fatalf("json reply: %v", err)
			}
			if len(r.Secrets["hello"]) == 0 { // XXX: depends on query
				b.Fatalf("unexpected reply: %s", stdout.String())
			}
		}()
	}
}

type reply struct {
	Secrets map[string]string
	Errors  []string
}
