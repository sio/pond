QEMU?=qemu-system-x86_64
KERNEL?=/boot/vmlinuz-$(shell uname -r)
INITRD?=/boot/initrd.img-$(shell uname -r)

.PHONY: boot
KERNEL_CMDLINE?=panic=-1
boot:  ## launch a diskless virtual machine with custom initramfs
	@echo -n "Kernel version "
	@LANG=C file $(KERNEL) | awk \
		'BEGIN { RS=", " } \
		 /version / { print substr($$0,9); found=1 } \
		 END { if (!found) { print "detection failed for $(KERNEL)"; exit 29; } }'
	$(QEMU) \
		-kernel $(KERNEL) \
		-initrd $(INITRD) \
		-append "console=tty0 console=ttyS0,115200 noquiet $(KERNEL_CMDLINE)" \
		-m 1G \
		-nographic \
		-chardev stdio,signal=off,mux=on,id=char0,logfile=console.log,logappend=on \
		-serial chardev:char0 \
		-mon chardev=char0 \
		-no-reboot \
		$(QEMU_ARGS)

.PHONY: demo
demo: build
demo:  ## boot our initramfs in qemu
	$(MAKE) run-mkinitrd boot \
		INIT=$(BIN_DIR)/init$(BIN_SUFFIX) \
		INITRD=bin/initramfs

.PHONY: demo-realtek
demo-realtek:  ## boot our initramfs with another network card
	$(MAKE) demo QEMU_ARGS="-netdev user,id=n0 -device rtl8139,netdev=n0"

include ../Makefile.golang

rand.bin: $(BIN_DIR)/rand$(BIN_SUFFIX)
	$< | dd of=$@ bs=1M count=50 status=progress iflag=fullblock

.PHONY: dieharder
dieharder: rand.bin  ## test our bespoke entropy generator
	dieharder -a -g 201 -f $< 2>&1 | tee $@.log
