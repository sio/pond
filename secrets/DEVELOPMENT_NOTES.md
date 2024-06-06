# Assorted development notes

## CLI UX mockup - not implemented yet

See [docs/cli_overview.md](docs/cli_overview.md) for the parts that have been
implemented already.


### Serving secrets

```console
$ secretd
$ secretd -l 127.0.0.1:2202 -C /path/to/secrets/root
```


### Fetching secrets

These queries are equivalent:

```console
$ SECRETS_HOST=secretd.example.com SECRETS_USER=/path/to/user-key.pub secrets /absolute/path relative/path
$ secrets -h secretd.example.com -u /path/to/user-key.pub /absolute/path relative/path
$ echo '["/absolute/path", "relative/path"]' | ssh secretd.example.com -i /path/to/user-key
$ echo '["/absolute/path", "relative/path"]' > query; secrets -f query
```

Successful output:

```
{
  "secrets: {
    "/absolute/path": "secret value from specific path",
    "relative/path": "secret value from one of available paths"
  },
  "errors": []
}
```

Error output:

```
{
  "secrets: {},
  "errors": [
    "certificate expired: username",
  ]
}
```

Convenient API for fetching secret to file:

```
$ scp secretd.example.com:path/to/secret local/file.txt
```


### Repo maintenance

```console
$ secretctl check
```

- Must not require any keys to execute
- Validate certificate chains for all users
- Validate all stored secrets
- Warnings for expired entries, errors for invalid ones


### Extend certificate lifetime

```
$ secretctl extend path/to/cert [90d]
```

- Check if original signer of the cert is available in ssh-agent
- Reissue the same cert with new "ValidBefore"
- Save to a new temporary file in the same directory
- Atomically replace original file with the new one while keeping a backup
  under incremented index


### Managing secrets

```
$ secretctl cp /source/path /destination/path
$ secretctl mv /source/path /destination/path
$ secretctl rm /secret/to/remove
```
