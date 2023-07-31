# pond/secrets: TODO list

## Low priority

- Make use of context.Context:
    - Set deadline
    - Use CancelFunc
- SSH client sometimes errors out while accessing API with either
  `client_loop: send disconnect: Broken pipe` or
  `client_loop: send disconnect: Connection reset by peer`.
  This happens after successfully writing API response.
  May be it's related to TCP connection life cycle on server?

## Lowest priority

- Try failure mode: ssh-agent dies on server and/or gets restarted
- Limit accepted public key algorithms. Should we? RSA pubkeys are pretty
  long, and we store them in database. Does this have any impact on
  performance.
  Even when not accepted, full public keys are written into logs - can this be
  used for denial of service? May be we should just send less data into logs,
  e.g. a fingerprint instead of the full key.
