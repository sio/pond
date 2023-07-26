include ../Makefile.golang
.DEFAULT_GOAL=run-secretd

# Not using static build here because of sqlite:
# sqlite3-binding.c:43784:
#   warning: Using 'dlopen' in statically linked applications requires at
#   runtime the shared libraries from the glibc version used for linking
#GOBUILD_ARGS=-tags netgo,osusergo -ldflags="-extldflags=-static"

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
