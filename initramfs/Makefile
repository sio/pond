KVM?=qemu-system-x86_64
KERNEL?=/boot/vmlinuz-$(shell uname -r)
INITRD?=/boot/initrd.img-$(shell uname -r)

.PHONY: boot
PANIC?=panic=-1
boot:  ## launch a diskless virtual machine with custom initramfs
	$(KVM) \
		-kernel $(KERNEL) \
		-initrd $(INITRD) \
		-append "console=tty0 console=ttyS0,115200 noquiet $(PANIC)" \
		-m 1G \
		-nographic \
		-no-reboot \

.PHONY: demo
demo: build
demo:  ## boot our initramfs in qemu
	$(MAKE) run-mkinitrd boot \
		INIT=$(BIN_DIR)/init$(BIN_SUFFIX) \
		INITRD=bin/initramfs

include ../Makefile.golang
