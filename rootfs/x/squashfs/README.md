# Simulate silent rootfs image corruption and detect it using dm-verity

Squashfs root filesystem images may get corrupted in transit and/or at rest.
Normally this corruption may go unnoticed for a long time, but with dm-verity
all image modifications will result in a loud i/o error.

Compare ouputs of `sudo make mount` and `sudo make mount-verity`.

By default corrupted squashfs image remains perfectly readable (if corruption did not
affect important filesystem metadata). Only manual comparison will show the difference:

```console
$ diff mnt/ok mnt/corrupt
Binary files mnt/ok/20.bin and mnt/corrupt/20.bin differ
                                                  ^^^^^^
                                          which implies that both
                                           files have been read
                                               successfully
```

With dm-verity corrupted data blocks become unreadable and trigger loud i/o
errors:

```console
$ diff mnt/ok mnt/corrupt
diff: mnt/corrupt/20.bin: Input/output error
                                       ^^^^^
                           nothing to compare to original,
                                   the data is gone
```

Here is how it looks in dmesg:

```dmesg
[ 3523.979841] device-mapper: verity: 7:1: data block 33280 is corrupted
[ 3523.980689] SQUASHFS error: Failed to read block 0x8200060: -5
[ 3523.981385] SQUASHFS error: Unable to read data cache entry [8200060]
[ 3523.982211] SQUASHFS error: Unable to read page, block 8200060, size 1020000
```
