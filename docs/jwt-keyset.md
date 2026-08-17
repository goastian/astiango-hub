# JWT Key Management

AstianGO Hub does not have a default JWT secret. The process requires a keyset injected by the environment's secret manager using one of these variables:

- `ASTIANGO_JWT_KEYSET`: The keyset JSON.

- `ASTIANGO_JWT_KEYSET_FILE`: The path to a mounted secret file containing the same JSON. This is the recommended option for Docker Swarm, Kubernetes, or Vault Agent.


Each secret must be Base64 encoded and at least 32 bytes (256 bits):

```json
{
"active_kid": "2026-09",
"keys": {
"2026-08": "<base64-of-a-key-of-32-bytes-or-more>",
"2026-09": "<base64-of-a-key-of-32-bytes-or-more>"

}
}
```

To rotate a key, first add the new key to the keyset and define it as `active_kid`. Retain the old keys for at least the maximum TTL of the token refresh plus the clock margin; then delete them. Retired keys immediately invalidate tokens signed with their `kid`.

The values ​​of `ASTIANGO_JWT_KEYSET` should never be stored in Git, images, or distributed configuration files.

For development compose, export it from your secrets manager before starting containers, for example: `export ASTIANGO_JWT_KEYSET="..."`.