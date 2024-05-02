# Generate some random data
data:
	mkdir -p $@
	for index in `seq 20`; do dd if=/dev/urandom bs=1M count=10 of=data/$$index.bin; done


# Build a file system
ok.squashfs: data $(wildcard data/*)
	mksquashfs $< $@ \
		-comp zstd \
		-noappend \
		-exit-on-error
	stat $@ --printf=%s > verity.offset
	/sbin/veritysetup format $@ $@ \
		--hash-offset=$(VERITY_OFFSET) \
		| tee verity.info
	/sbin/veritysetup verify $@ $@ $(VERITY_HASH) \
		--hash-offset=$(VERITY_OFFSET)
VERITY_OFFSET=$$(cat verity.offset)
VERITY_HASH=$$(awk '/Root hash/ { print $$3; }' verity.info)


# Simulate silent data corruption
corrupt.squashfs: ok.squashfs
	cp $< $@
	printf CORRUPT | dd of=$@ bs=1 seek=$$((0x8200040)) conv=notrunc # corruption near the end of the last file


.PHONY: build
build: ok.squashfs corrupt.squashfs


MOUNTPOINTS=mnt/ok mnt/corrupt
$(MOUNTPOINTS):
	mkdir -p $@


# Mount file systems for inspection.
# Corrupted image mounts fine because we made sure not to mess with squashfs
# metadata when simulating corruption.
.PHONY: mount
mount: ok.squashfs corrupt.squashfs $(MOUNTPOINTS) | umount
	mount ok.squashfs mnt/ok
	mount corrupt.squashfs mnt/corrupt
	$(MAKE) compare


# Mount with data verification
.PHONY: mount-verity
mount-verity: ok.squashfs corrupt.squashfs $(MOUNTPOINTS) | umount
	veritysetup open ok.squashfs ok ok.squashfs $(VERITY_HASH) --hash-offset=$(VERITY_OFFSET)
	veritysetup open corrupt.squashfs corrupt corrupt.squashfs $(VERITY_HASH) --hash-offset=$(VERITY_OFFSET)
	mount /dev/mapper/ok mnt/ok
	mount /dev/mapper/corrupt mnt/corrupt
	$(MAKE) compare


# Compare mounted filesystems
.PHONY: compare
compare:
	-sha256sum $(patsubst %,%/*,$(MOUNTPOINTS)) | sort -t/ -k3 -h
	diff -s $(MOUNTPOINTS)


.PHONY: umount
umount:
	-umount mnt/ok
	-umount mnt/corrupt
	-veritysetup close ok
	-veritysetup close corrupt


.PHONY: clean
clean: umount
	git clean -idx


.PHONY: apt
apt:
	apt install -y --no-install-recommends squashfs-tools cryptsetup-bin


.PHONY: ci update upx
ci update upx:  ## noop, required for `make ci` at repo level
