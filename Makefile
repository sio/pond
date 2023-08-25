.PHONY: ci
ci:
	@for dir in */; do $(MAKE) -C $$dir ci; done
