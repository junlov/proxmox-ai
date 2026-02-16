# Plan ID Rules

## Purpose

Define how `plan_id` values are created, validated, expired, and consumed.

## Generation

- `plan_id` is generated only by `POST /v1/actions/plan`.
- IDs must be collision-resistant and opaque (for example UUIDv7 or equivalent).
- Clients must treat `plan_id` as an immutable reference to a single planned request.

## Binding and integrity

- Each `plan_id` is bound to:
  - normalized action payload hash
  - environment
  - policy decision snapshot
  - creation timestamp
- `apply` must provide `plan_id` and a payload that hash-matches the planned payload.

## Expiration

- Plans have a TTL.
- Default TTL target: 15 minutes for high-risk actions, 60 minutes for low/medium (configurable).
- Expired plans are not apply-eligible and transition to `expired`.

## Consumption

- `plan_id` is single-consumption for successful mutation semantics.
- First successful `apply` consumes the plan (`applied` terminal state).
- Replays after terminal states are rejected; caller must request a new plan.

## Error behavior

- Unknown `plan_id`: not found error.
- Expired `plan_id`: expired error.
- Hash mismatch: integrity error.
- Reused consumed `plan_id`: replay error.

## Audit requirements

- Audit records must include `plan_id`, decision summary, and final apply outcome.
- Denied and expired paths must still emit audit records.
