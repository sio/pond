# pond/secrets: TODO list

## Medium priority

- Review and document seed entropy size in all cryptographic operations to avoid
  mistakes similar to [Milk Sad] vulnerability

[Milk Sad]: https://news.ycombinator.com/item?id=37054862

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
