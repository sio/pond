# secretd usage

<!--SECTION bin/secretd@linux-amd64 --help START OFFSET 1-->
```console
$ bin/secretd@linux-amd64 --help
Usage: secretd@linux-amd64

Flags:
  -h, --help              Show context-sensitive help.
  -C, --chdir=path        Change working directory prior to executing
                          ($SECRETS_DIR)
  -l, --listen=address    Address for secretd to bind to, e.g.
                          tcp://10.0.0.123:345 or unix:///var/run/secretd.socket
                          (default: tcp://127.0.0.1:20002) ($SECRETS_BIND)
```
<!--SECTION bin/secretd@linux-amd64 --help END OFFSET 1-->
