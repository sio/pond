# secretctl usage

## List of subcommands

<!--SECTION bin/secretctl@linux-amd64 --help START OFFSET 1-->
```console
$ bin/secretctl@linux-amd64 --help
Usage: secretctl@linux-amd64 <command>

Flags:
  -h, --help          Show context-sensitive help.
  -C, --chdir=path    Change working directory prior to executing ($SECRETS_DIR)

Commands:
  init <pubkey>
    Initialize secrets repository in an empty directory

  cert --user=name --key=path <prefix> ...
    Issue certificate to delegate user/administrator privileges

  set <secret> [<value>]
    Set secret value from argument/file/stdin/$EDITOR

Run "secretctl@linux-amd64 <command> --help" for more information on a command.
```
<!--SECTION bin/secretctl@linux-amd64 --help END OFFSET 1-->


## Repo initialization

<!--SECTION bin/secretctl@linux-amd64 init --help START OFFSET 1-->
```console
$ bin/secretctl@linux-amd64 init --help
Usage: secretctl@linux-amd64 init <pubkey>

Initialize secrets repository in an empty directory

Arguments:
  <pubkey>    Path to public part of ssh keypair to be used as repository master
              key

Flags:
  -h, --help          Show context-sensitive help.
  -C, --chdir=path    Change working directory prior to executing ($SECRETS_DIR)
```
<!--SECTION bin/secretctl@linux-amd64 init --help END OFFSET 1-->


## Access management

<!--SECTION bin/secretctl@linux-amd64 cert --help START OFFSET 1-->
```console
$ bin/secretctl@linux-amd64 cert --help
Usage: secretctl@linux-amd64 cert --user=name --key=path <prefix> ...

Issue certificate to delegate user/administrator privileges

Arguments:
  <prefix> ...    List of path prefixes to delegate privileges over

Flags:
  -h, --help             Show context-sensitive help.
  -C, --chdir=path       Change working directory prior to executing
                         ($SECRETS_DIR)

  -u, --user=name        Human readable user identifier
  -a, --admin=name       Human readable administrator identifier
  -k, --key=path         Public key of recipient
  -r, --read             Read access flag
  -w, --write            Write access flag
  -x, --expires="90d"    Certificate validity duration (default: 90d)
```
<!--SECTION bin/secretctl@linux-amd64 cert --help END OFFSET 1-->
