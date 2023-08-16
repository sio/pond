# pond/secrets: TODO list

## High priority: blocks deployment

- Implement secretctl
- Implement secretd


## Medium priority: quality of life

- Write shell-based CLI UX tests: source test-setup.sh; set -v -x; run;
  compare with saved output
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
- SSH client sometimes errors out while accessing API with either
  `client_loop: send disconnect: Broken pipe` or
  `client_loop: send disconnect: Connection reset by peer`.
  This happens after successfully writing API response.
  May be it's related to TCP connection life cycle on server?


## Lowest priority: maybe sometime (if ever)

- Try failure mode: ssh-agent dies on server and/or gets restarted
