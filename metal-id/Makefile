.DEFAULT_GOAL=run-metal-id

export CGO_ENABLED=0
GOBUILD_ARGS=-ldflags="-s -w"
GOBUILD_MUST_STATIC=yes

include ../Makefile.golang

.PHONY: ci
ci: lint test build run-metal-id

run-metal-id: ARGS=-verbose
