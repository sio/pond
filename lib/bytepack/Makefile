include ../../Makefile.golang

.PHONY: ci
ci: lint test

.PHONY: fuzz
fuzz:
	$(GO) test ./tests -fuzz=Fuzz

.PHONY: bench
bench:
	$(GO) test ./tests -bench=Benchmark -benchmem -run='^#'
