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
- End-to-end encryption is not a goal, because we want shared secrets by
  design. Users writing secrets values and users reading secrets are
  most often not the same. In such scenario we would either need to hardcode
  public keys of all readers when saving the secret or (as we do now) save
  secrets under server master key, so non-E2EE.
- Untrusted server administrator is not a threat we consider, even though we
  try to make it somewhat harder for an attacker to exfiltrate secrets even if
  both database and the private keys are leaked.


## API

- Rate limiting is out of scope. Use existing firewall solutions for that.
- One API request per SSH session. Multiple values may be read/writted in one
  API request (within reason).


## Users and permissions

- Each key may belong to one and only one user
- User account classes:
    - Regular users may only read/write secrets according to their assigned
      roles via SSH API
    - Administrators may manage access control lists, add new users, assign
      roles via SSH API
    - Server administrators who have access to full journal file(s) may get a
      full overview of existing secrets (without values), user accounts and
      access control lists. This functionality is not exposed through API.
      Full journal file is not required for day to day operation and is better
      not stored on the same server where secretd is running.
- Administrator accounts may not be used to access (read or write) any secret values.
  This is a convenience and maintenance feature, not a security measure
  because administrators may simply add new user accounts with any privileges.
