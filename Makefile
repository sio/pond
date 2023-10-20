# Execute common actions across all subprojects in monorepo. Fail at first error.
ACTIONS=ci update clean build upx

.PHONY: $(ACTIONS)
$(ACTIONS):
	@find \
		-mindepth 2 \
		-name Makefile \
		! -execdir \
			$(MAKE) $@ \; \
		-exec false {} + \
		-quit
