# Expermienting with TLS certificates

## Goal

Establish a secure NBD connection with mutual authentication between client
and server based on preshared SSH keys:

- Server has a list of public keys of clients that are allowed to connect
  (similar to `~/.ssh/authorized_keys`)
- Clients have a list of public keys of servers that they trust
  (similar to `~/.ssh/known_hosts`)

## Overview

Client and server implement PKI in a similar fashion:

- SSH key is used to issue root CA certificate
- Ephemeral certificates are issued for all allowed counterparties
  (can we do this knowing only their public key, without a signed CSR?)
- These certificates from a CA store for establishing TLS connections
- Ephemeral certificate issued by our root CA is used to represent our side in
  TLS handshake
