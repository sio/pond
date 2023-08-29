.PHONY: ci
ci:
	@find \
		-mindepth 2 \
		-name Makefile \
		! -execdir \
			$(MAKE) ci \; \
		-exec false {} + \
		-quit
