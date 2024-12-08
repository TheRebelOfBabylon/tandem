# tandem
Nostr relay, written in Go.

# NIP Compliance
- [x] NIP-01
- [x] NIP-02: Follow List
- [ ] NIP-05: Mapping Nostr keys to DNS-based Internet Identifiers
- [ ] NIP-09: Event Deletion Request
- [ ] NIP-11: Relay Information Document
- [ ] NIP-13: Proof of Work
- [ ] NIP-17: Private Direct Messages
- [ ] NIP-29: Relay-based Groups
- [ ] NIP-40: Expiration Timestamp
- [ ] NIP-42: Authentication of clients to relays
- [ ] NIP-45: Event Counts
- [x] NIP-50: Search Capability*
- [ ] NIP-56: Reporting
- [ ] NIP-64: Chess (Portable Game Notation)
- [x] NIP-65: Relay List Metadata**
- [ ] NIP-70: Protected Events
- [ ] NIP-86: Relay Management API
- [ ] NIP-96: HTTP File Storage Integration

* = Search is not ordered by quality or treated differently for kind. Applies only to content field and no special syntax is used (not even * for wildcard)
** = tandem does not currently disallow any users from submitting lists

# Goals

- Easy to deploy: anyone's Uncle Jim with any sliver of IT knowledge should be able to deploy a relay.
- Easy to moderate: blocking IPs, whitelisting pubkeys, setting up specific moderation rules, all should be achievable without code knowledge.
- Community driven: nostr's main usecase is as a global townsquare but it can also be used to create small communities. Tandem's main goal is to serve the latter usecase. 

# Roadmap

- [ ] Define a roadmap

# Usage

## Prerequisites
- [edgedb](https://www.edgedb.com/)

## Installation

Define a configuration file:
```toml
[http]
host=localhost # env var: HTTP_HOST
port=5150 # env var: HTTP_PORT

[log]
level=info # env var: LOG_LEVEL, one of debug|info|error, default: info
log_file_path=/path/to/file.log # env var: LOG_FILE_PATH, optional 

[storage]
uri="edgedb://edgedb:<password>@localhost:10701/main" # env var: STORAGE_URI, replace with your edgedb credentials, one of edgedb|memory
skip_tls_verify=true # env var: STORAGE_SKIP_TLS_VERIFY, default: false
```

Run:
```shell
$ tandem -config <path_to_toml_file>
```

# Tests

with edgedb
```shell
$ STORAGE_URI=<your_uri> go test -v -tags=storage,edgedb ./...
```

with memorydb
```shell
$ STORAGE_URI="memory://" go test -v -tags=storage,memory ./...
```

without storage
```shell
$ go test -v ./...
```
