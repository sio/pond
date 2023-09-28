# pond/secrets: TODO list

## High priority: blocks deployment

- Implement secretctl
- Implement secretd


## Medium priority: quality of life

- Write documentation
    - Security model
    - Storage model
    - Threat model
    - Usage
- Review and document seed entropy size in all cryptographic operations to avoid
  mistakes similar to [Milk Sad] vulnerability

[Milk Sad]: https://news.ycombinator.com/item?id=37054862


## Low priority: nice to have

- Make use of context.Context:
    - Set deadline
    - Use CancelFunc
- Provide tools to rekey all secrets and certificates to a new master key


## Lowest priority: maybe sometime (if ever)

- Try failure mode: ssh-agent dies on server and/or gets restarted
- Multipath secrets: save the same value to multiple paths and enforce that
  these paths are manupulated together from now on (modify, remove)
- Re-sign secrets by master key with unique decryption key (derived from nonce
  and master signature). Is there a point in doing this given that we expect
  old version to be stored in the same git repo anyways?
