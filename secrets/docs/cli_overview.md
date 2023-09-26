# CLI overview

## Initializing secrets repository

Secrets repository is a directory where all secret values are stored
(in encrypted form) along with required access control information.

```console
$ secretctl init /path/to/master-key.pub
```

This command will:

- Check if current directory is empty
- Query ssh-agent for the listed key
- Create directory structure for secrets repo
- Generate master key certificate and save it to the repo


## Access control

Access is granted via certificates:

- Master certificate is self signed by master key
- Master key signs administrator certificates, administrator keys are used
  to sign user certs
- All certificate chains follow this three tier model, arbitrarily long cert
  chains are not supported
- Only certificates stored in the repository are being considered when
  checking user privileges, supplying certificates at connection time is not
  supported
- More specific paths are considered more restricted, e.g:
    - To read `/secret` users need to have read privileges for any path under `/`
    - Users with read access to `/test/servers/` will be able to read
      `/test/servers/secret`, `/test/secret` and `/secret`, but will not be
      able to read `/test/servers/mail/secret` or `/prod/secret`

```console
$ secretctl cert --user alice --key /path/to/alice-key.pub --read --write /first/path /second/path
$ secretctl cert -u alice -k /path/to/alice-key.pub -rw /first/path /second/path
$ secretctl cert -u bob -k /path/to/bob-key.pub -r /bobs/readonly/path
$ secretctl cert --admin root -k /path/to/admin-key.pub -rw /
$ secretctl cert -a charlie -k /path/to/charlie-key.pub -r /specific/prefix
```

These commands will:

- Query ssh-agent for a key that has enough privileges for required action
- Generate user/administrator certificate and save it to the secrets repo


## Writing secrets

Users with write permissions to specified paths can set secret values by
providing them on command line, from file, from standard input (pipe) or by
launching interactive $EDITOR and saving opened file.

```console
$ secretctl set /path/to/first-secret "literal-value"
$ secretctl set /path/to/second-secret -f /from/file.txt
$ cat /from/anywhere.txt | secretctl set /path/to/third-secret
$ secretctl set /path/to/fourth-secret  # opens in $EDITOR
$ secretctl set /path/to/short-lived-secret "literal-value" --expires=1d12h31m
```

These commands will save provided values into encrypted files under
repository root. Only master key can decrypt those values after saving.
This allows us to use Git for tracking secrets (even though encrypted diffs
are less meaningful) and for collaboration: less privileged users may send
pull requests for changing the parts they are allowed to change.
