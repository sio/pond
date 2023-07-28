# Specification

> This document does not live up to its name and probably never will.
> Nevertheless, it's nice to have high aspirations :)
>
> For now consider it to be a list of guidelines for developer(s).


## Encryption

- Each secret is encrypted with an unique key derived from master keypair
- Private key from master keypair is never touched by our software.
  We delegate that to ssh-agent.
- Unencrypted secret values are never saved to any storage and are deleted
  from memory as early as possible after processing.


## API

- Rate limiting is out of scope. Use existing firewall solutions for that.
- One API request per SSH session. Multiple values may be read/writted in one
  API request (within reason).


## Users and permissions

- Each key may belong to one and only one user
- Administrator accounts may not be used to access (read or write) any secret values
