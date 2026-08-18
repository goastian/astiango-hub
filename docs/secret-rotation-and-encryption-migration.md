# Secret Rotation and Encryption Migration

AstianGO Hub does not include a default administrative account, authentication key, or encryption key. These must be provided to the process from a secret manager using environment variables or an equivalent injection mechanism.

## First Boot

Before starting an empty control plane, inject the following unique values:

- `ASTIANGO_JWT_KEYSET`: JWT keyset already described in [keyset management](jwt-keyset.md).

- `ASTIANGO_AUTH_KEY`: Shared secret of at least 32 bytes for internal authentication between master and workers. All nodes must receive the same value.

- `ASTIANGO_ENCRYPTION_KEYSET`: JSON with `active_kid` and Base64 keys of exactly 32 bytes. Structural example: `{"active_kid":"2026-09","keys":{"2026-09":"<base64-32-bytes>"}}`.

`ASTIANGO_BOOTSTRAP_ADMIN_USERNAME` and `ASTIANGO_BOOTSTRAP_ADMIN_PASSWORD`: unique credentials for the first administrator; the password must be at least 12 characters long.

The bootstrap is only consumed if a root admin does not exist, and the created account is forced to change its password. After creating and verifying the account, remove both bootstrap variables from the deployment. Do not keep them in `.env` files, images, or logs.

For existing installations, create or rotate an administrator account with an authorized operator before upgrading. Review accounts that still have `must_change_password: true`, change their passwords, and revoke their sessions before exposing the new version.

## Rotating Authentication Keys and JWT

1. Generate new values ​​in the secrets manager.

2. For JWT, publish a keyset that retains the old `kid` and appends the new one as `active_kid`; wait for the old tokens to reach their maximum expiration and then retire the old one.

3. For `ASTIANGO_AUTH_KEY`, deploy the master and workers together with the new value. The protocol uses a single shared key, so it does not support mixing values ​​during the deployment window.

4. Verify health checks and internal authentication; revoke the old secret in the secrets manager.

## AES-256-GCM and Cipher Key Rotation

The new values ​​are stored as `agcm:v1:<kid>:<payload>`. The payload contains a random GCM nonce followed by the ciphertext and its authentication tag. The header and `kid` are bound as AAD; any modification is rejected before returning plaintext.

To rotate the encryption key:

1. Add a new 32-byte key to the keyset and make it `active_kid`, keeping the old keys for reading.

2. Restart the processes to receive the updated keyset.

3. Perform a verifiable backup of MongoDB and first run the report: `astiango-hub migrate-encryption`.

4. Run `astiango-hub migrate-encryption --apply`. This command idempotently converts `databases.encrypted_password`; current values ​​remain intact.

5. Validate connections and remove the old keys only after the report indicates zero legacy values ​​and the backups have expired according to the retention policy.

## Migrate legacy CBC data

The CBC reader exists exclusively for migration. Temporarily inject `ASTIANGO_ENCRYPTION_LEGACY_KEY` with the Base64-encoded historical key alongside the new AES-GCM keyset, run the previous steps, and remove the variable immediately upon completion. If this variable is missing, the process will reject legacy ciphertexts; there is no compatibility key compiled in the binary.