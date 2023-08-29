include ../../Makefile.golang

.PHONY: ci
ci: lint test

.PHONY: render-test-samples
render-test-samples:
	$(GO) test ./test --tags=write_rendered_samples --count=1 -v
