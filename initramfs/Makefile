KVM?=qemu-system-x86_64
KERNEL?=/boot/vmlinuz-$(shell uname -r)
INITRD?=/boot/initrd.img-$(shell uname -r)

.PHONY: boot
boot:  ## launch a diskless virtual machine with custom initramfs
	$(KVM) \
		-kernel $(KERNEL) \
		-initrd $(INITRD) \
		-append "console=tty0 console=ttyS0,115200 noquiet panic=-1" \
		-m 1G \
		-nographic \
		-no-reboot \

include ../Makefile.golang
