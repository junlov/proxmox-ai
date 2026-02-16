# Runtime Contract: pi agent and proxmox-agent

## Purpose

Define a strict contract where `pi agent` is the primary orchestrator and `proxmox-agent` is the safe execution boundary for Proxmox operations.

Related ADRs:

- `docs/adr/0001-control-plane-execution-boundary.md`
- `docs/adr/0002-home-cloud-credential-isolation.md`

## Roles

- `pi agent`
  - Interprets user intent.
  - Selects and sequences concrete actions.
  - Calls plan and apply endpoints.
  - Collects and attaches required approvals for high-risk operations.
- `proxmox-agent`
  - Validates action payloads.
  - Evaluates risk and approval policy.
  - Executes approved actions through the Proxmox client.
  - Records audit events for plan, deny, and apply outcomes.

## Required flow

1. Submit `POST /v1/actions/plan`.
2. Inspect returned decision and risk level.
3. If high risk, require `approved_by` before apply.
4. Submit `POST /v1/actions/apply`.
5. Persist and review audit record.

Lifecycle states and transitions are defined in `docs/plan-apply-lifecycle.md`.
`plan_id` issuance, TTL, and consumption semantics are defined in `docs/plan-id-rules.md`.
Plan/apply payload integrity validation is defined in `docs/request-hashing.md`.
Idempotency key behavior is defined in `docs/idempotency.md`.

## Safety invariants

- No direct state-changing execution outside of the plan/apply flow.
- High-risk actions (`delete_vm`, `migrate_vm`, `storage_edit`, `firewall_edit`) require explicit approval metadata.
- Home and cloud environments are distinct trust boundaries.
- TLS verification must remain enabled in production environments.
- Token secrets must never be logged or returned by the API.
