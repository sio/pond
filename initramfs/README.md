# Bespoke initramfs for pond machines

Both dracut and initramfs-tools offer catch-all solutions and present
themselves as rat's nests of shell scripts. This is likely unavoidable given
their scope and their limitations but in our usecase we can do better. This
project provides tools to create and execute a small initramfs which mounts
root filesystem over NBD protocol.

See [components.dot](components.dot) for an overview of what our initramfs does.

## Scratchpad: developer's notes to self

- Launch all boot stages at once. Goroutines are cheap. Let Go scheduler do
  the dirty work for us.
- Many goroutines will be waiting for a dependency stage to finish: use Go
  channels (closed = done; open = waiting; map[stage]chan with locking)
- Use a mutex when writing to stdout: do not output log messages when an
  interactive menu is in use. Save logs to bytes.Buffer to unblock callers
  immediately, and flush to stdout after user closes the menu.
- Compressing binaries with UPX prior to including them into initramfs is
  meaningless because we will later compress initramfs archive with zstd
  anyways. Total image size reduction is close to zero.
