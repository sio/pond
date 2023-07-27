# pond/secrets: TODO list

## Low priority

- Limit max size of incoming API request to 16-20KB (max payload size
  recommended for nacl.SecretBox)
    - In body reader?
    - In JSON deserialization routine?
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
