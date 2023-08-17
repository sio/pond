# Assorted development notes

## CLI UX mockup

### Initializing secrets repository

```console
$ secretctl init /path/to/master-key.pub
```

- Check if current directory is empty
- Check if master key is loaded into ssh-agent
- Create directory structure
- Generate master key certificate


### Issuing certificates for users/admins

```console
$ secretctl user -n alice -k /path/to/alice-key.pub -rw /first/path /second/path
$ secretctl user -n bob -k /path/to/bob-key.pub -r /bobs/readonly/path
$ secretctl admin -n root -k /path/to/admin-key.pub -rw /
$ secretctl admin -n charlie -k /path/to/charlie-key.pub -r /specific/prefix
```

- Check if ssh-agent contains a key that is allowed to delegate capabilities:
    - For issuing admin certs: master key
    - For issuing user certs: admin key with proper set of caps and paths
    - Try all certs from ssh-agent until one fits or none left to try
- Generate a certificate
- Save to
    - $root/access/user/$name-$index.cert
    - $root/access/admin/$name-$index.cert


### Writing secrets

```console
$ secretctl set /path/to/secret -v "literal-value"
$ secretctl set /path/to/secret -f /from/file.txt
$ cat /from/anywhere.txt | secretctl set /path/to/secret
$ secretctl cp /source/path /destination/path
$ secretctl mv /source/path /destination/path
$ secretctl rm /secret/to/remove
```


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
