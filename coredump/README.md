# Steps for debugging stripped binaries

  1. Reproduce release build bit-perfectly: `make build`, check hash sum
  2. Rebuild with debug symbols: `make build DEBUG=y`
  3. Use debug binaries with coredump from release build:
     `make debug-core CORE=dump.file EXE=bin/whatever@os-arch.debug`
