# Guestauth

The `guestauth` service gives guest users access to a share without a full
OpenCloud account. When a share is created for a user of type
`USER_TYPE_GUEST`, the service issues a one-time invitation token; redeeming
that token exchanges it for a signed session cookie that authenticates the
guest.

It is part of the default service set and does not need to be enabled with
`OC_ADD_RUN_SERVICES`.

## Overview

- **Consumes** the share lifecycle events `ShareCreated`, `ShareRemoved` and
  `ShareExpired`.
- **Publishes** the `GuestTokenCreated` event carrying the invitation token,
  so the invitation can be delivered to the guest.
- Exposes an unauthenticated endpoint that redeems the token and sets a
  session cookie.
- Stores only hashes of the token and deletes the stored record when the share
  is removed or expires.

## Token lifecycle

1. **Issue** — on the consumed `ShareCreated` event, where the grantee is a
   guest, the service generates a random secret and stores a record keyed by
   the hash of the share id. It then publishes the `GuestTokenCreated` event
   with the token.
2. **Redeem** — the guest posts the token to
   `POST /graph/v1beta1/guestInvitations/redeem`. The service validates the
   token and the share, marks the token as used and returns a signed JWT
   session token in a cookie. Tokens are single-use.
3. **Cleanup** — on the consumed `ShareRemoved` or `ShareExpired` event, the
   stored record is deleted.

## Configuration

The service is configured via `GUESTAUTH_*` environment variables or a
`guestauth.yaml` file.

To run only the HTTP part, set `GUESTAUTH_EVENTS_DISABLED=true`. To run only
the event consumer, set `GUESTAUTH_HTTP_DISABLED=true`.

Relevant options:

- `GUESTAUTH_JWT_SECRET` — secret used to sign session tokens.
- `GUESTAUTH_JWT_COOKIE_NAME`, `GUESTAUTH_JWT_TTL` — session cookie name and
  lifetime.
- `GUESTAUTH_TOKENS_STORAGE_ROOT` — where invitation token records are stored.
- `GUESTAUTH_SERVICE_ACCOUNT_ID`, `GUESTAUTH_SERVICE_ACCOUNT_SECRET` — service
  account used to query the gateway for share metadata.
- `GUESTAUTH_NUM_CONSUMERS` — number of concurrent event consumers.
- `OC_REVA_GATEWAY` — CS3 gateway used to look up shares.
