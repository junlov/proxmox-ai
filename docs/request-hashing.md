# Request Hashing and Integrity Validation

## Purpose

Define deterministic hashing so `apply` can verify it is executing exactly what was planned.

## Canonicalization

- Build a canonical JSON object containing:
  - `environment`
  - `action`
  - `target`
  - `params` (object keys sorted recursively)
  - `dry_run`
- Exclude approval metadata (`approved_by`) from payload hash to preserve policy checks separately.

## Hash algorithm

- Use SHA-256 over UTF-8 canonical JSON bytes.
- Encode as lowercase hex string.
- Store hash with each plan record.

## Validation rules

- `plan` computes and stores `request_hash`.
- `apply` recomputes hash from submitted request payload fields.
- If hashes differ, reject apply with integrity violation.

## Error contract

- `code`: `REQUEST_HASH_MISMATCH`
- `reason`: `apply payload does not match planned payload`
- include `plan_id` and correlation ID in error response.

## Security notes

- Hashing validates request integrity, not actor authorization.
- Hash values are safe to log; raw secrets and token values are not.
