.DEFAULT_GOAL=run-metal-id

include ../Makefile.golang

.PHONY: ci
ci: lint test build run-metal-id

run-metal-id: ARGS=-verbose
