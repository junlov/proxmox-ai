# Idempotency Key Semantics

## Purpose

Define duplicate request handling for `plan` and `apply` endpoints.

## Key requirements

- Clients send an idempotency key per mutation attempt (header or request metadata as defined by API).
- Keys are scoped to endpoint and caller identity.
- Keys must be unique for semantically distinct operations.

## Behavior

- First request with a new key executes normally and stores response metadata.
- Repeat request with same key and same canonical payload returns original response.
- Repeat request with same key and different payload returns conflict.

## Conflict contract

- `code`: `IDEMPOTENCY_CONFLICT`
- `reason`: `idempotency key reused with different payload`
- include correlation ID for debugging.

## Retention

- Idempotency records have TTL and are garbage-collected after expiry.
- Expired keys are treated as new keys.

## Notes

- Idempotency does not replace plan/apply integrity checks.
- High-risk approvals are still evaluated on apply.
