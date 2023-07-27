.DEFAULT_GOAL=run-secretd


# https://www.arp242.net/static-go.html
GOBUILD_ARGS=-tags netgo,osusergo,sqlite_omit_load_extension -ldflags="-extldflags=-static"
GOBUILD_MUST_STATIC=yes


# Fail if C compiler is not available (required for sqlite)
export CGO_ENABLED=1


$(EXE): $(shell find . -type f -iname '*.sql')


MANUAL_TEST_PORT=2222
MANUAL_TEST_SOCKET=/tmp/pond-secrets-test.socket
MANUAL_TEST_KEY=tests/keys/storage
export SSH_AUTH_SOCK=$(MANUAL_TEST_SOCKET)
manual-test-agent:
	setsid ssh-agent -a $(MANUAL_TEST_SOCKET)
	chmod go-rw $(MANUAL_TEST_KEY)
	ssh-add $(MANUAL_TEST_KEY)


manual-test-client:
	-echo Default SSH client    | ssh localhost -p $(MANUAL_TEST_PORT)
	-echo No-PTY SSH client     | ssh localhost -p $(MANUAL_TEST_PORT) -T
	-echo Exec command over SSH | ssh localhost -p $(MANUAL_TEST_PORT) command
	-echo Exec + no-pty client  | ssh localhost -p $(MANUAL_TEST_PORT) -T command


include ../Makefile.golang
