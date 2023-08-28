.DEFAULT_GOAL=run-secretd


# https://www.arp242.net/static-go.html
GOBUILD_ARGS=-tags netgo,osusergo,sqlite_omit_load_extension -ldflags="-extldflags=-static"
GOBUILD_MUST_STATIC?=yes


# Fail if C compiler is not available (required for sqlite)
export CGO_ENABLED=1


$(EXE): $(shell find . -type f -iname '*.sql')


.PHONY: test-cli
test-cli: GOTEST_ARGS+=--tags=test_cli
test-cli: build
	$(GO) test ./tests/cli -timeout $(GOTEST_TIMEOUT) $(GOTEST_ARGS)


.PHONY: test-docs
test-docs: GOTEST_ARGS+=--tags=docs
test-docs: build
	$(GO) test ./tests/help -timeout $(GOTEST_TIMEOUT) $(GOTEST_ARGS)
	git diff --exit-code -- docs/


.PHONY: ci
ci: lint test test-cli


.PHONY: review-crypto-algorithms
review-crypto-algorithms:
	@git grep 'golang.org/x/crypto'|grep -v path_algo_test.go|grep -vE '^go\.'|grep -v 'crypto/ssh'|column -t


.PHONY: git-config
git-config:
	git config --local diff.ssh-certificate.textconv "ssh-keygen -Lf"
	git config --local diff.ssh-certificate.cachetextconv true


include ../Makefile.golang
