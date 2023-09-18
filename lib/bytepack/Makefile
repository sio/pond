include ../../Makefile.golang

.PHONY: fuzz
fuzz:
	$(GO) test ./tests -fuzz=Fuzz

.PHONY: bench
bench:
	$(GO) test ./tests -bench=Benchmark -benchmem -run='^#'
