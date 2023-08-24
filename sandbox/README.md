# Lightweight sandbox environment for Linux

This Golang library provides a thin wrapper around `unshare` from util-linux
that allows to execute series of commands in a lightweight sandboxed
environment using Linux kernel namespaces.

No security guarantees are provided. This tool was created to simplify testing
command line applications in a minimal environment: to surface errors in case
if any unexpected implicit dependencies or permissions are required.
