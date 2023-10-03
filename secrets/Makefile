.DEFAULT_GOAL=run-secretd


# https://www.arp242.net/static-go.html
GOBUILD_ARGS=-tags netgo,osusergo,sqlite_omit_load_extension -ldflags="-extldflags=-static -s -w"
GOBUILD_MUST_STATIC?=yes


# Fail if C compiler is not available (required for sqlite)
export CGO_ENABLED=1


$(EXE): $(shell find . -type f -iname '*.sql')


.PHONY: test-cli
test-cli: GOTEST_ARGS+=--tags=test_cli
test-cli: build
	$(GO) test ./tests/cli -timeout $(GOTEST_TIMEOUT) $(GOTEST_ARGS)


.PHONY: test-docs
test-docs: GOTEST_ARGS+=--tags=docs --count=1
test-docs: build
	$(GO) test ./tests/help -timeout $(GOTEST_TIMEOUT) $(GOTEST_ARGS)
	git diff --exit-code -- docs/


.PHONY: ci
ci: lint test test-cli test-docs


.PHONY: review-crypto-algorithms
review-crypto-algorithms:
	@git grep 'golang.org/x/crypto'|grep -v path_algo_test.go|grep -vE '^go\.'|grep -v 'crypto/ssh'|column -t


.PHONY: git-config
git-config:
	git config --local diff.ssh-certificate.textconv "ssh-keygen -Lf"
	git config --local diff.ssh-certificate.cachetextconv true


.PHONY: manual-test-prepare
manual-test-prepare:
	chmod go-rwx tests/keys/*
	ssh-add tests/keys/master tests/keys/alice tests/keys/bob
	bin/secretctl$(BIN_SUFFIX) init tests/keys/master.pub
	bin/secretctl$(BIN_SUFFIX) cert --admin=alice --key=tests/keys/alice.pub -rw / /users/alice
	bin/secretctl$(BIN_SUFFIX) cert --user=bob --key=tests/keys/bob.pub -rw /users
	bin/secretctl$(BIN_SUFFIX) set /users/domain users.domain.tld
	bin/secretctl$(BIN_SUFFIX) set /domain top.domain.tld

.PHONY: bench-stress
bench-stress: SECRETD_BENCH_SERVER?=tcp://localhost:20002
bench-stress: SECRETD_BENCH_CLIENT_KEY?=$(abspath tests/keys/bob)
bench-stress: SECRETD_BENCH_QUERY?=foo,bar
bench-stress: GOBENCH_TIME?=10s
bench-stress:
	go test ./tests/stress -bench=. -benchmem -benchtime=$(GOBENCH_TIME) -timeout=15s$(GOBENCH_TIME)
export SECRETD_BENCH_SERVER
export SECRETD_BENCH_CLIENT_KEY
export SECRETD_BENCH_QUERY


include ../Makefile.golang
