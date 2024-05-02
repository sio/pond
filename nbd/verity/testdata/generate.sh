#!/bin/bash
set -euo pipefail
set -x

# Generate pseudorandom data for our filesystem.
# Use static seed to provide transparency against xz-style attack
# (https://en.wikipedia.org/wiki/XZ_Utils_backdoor)
temp=$(mktemp --directory --tmpdir "pond-rnd-XXXXXXXX")
go run ./generate_files.go "${temp}" 20 10000

# Create squashfs image
image=pseudorandom.squashfs
mksquashfs "${temp}" "${image}" \
    -comp zstd \
    -noappend \
    -exit-on-error

# Append verity superblock and hash tree
offset=$(stat "${image}" --printf=%s)
PATH="$PATH:/sbin:/usr/sbin"
veritysetup format "${image}" "${image}" --hash-offset="${offset}" | tee "${image%.*}.verity"

# Save file checksums
pushd "${temp}"
sha256sum * | tee ~1/"${image%.*}.checksum"
popd

# Clean up
rm -rf "${temp}"
