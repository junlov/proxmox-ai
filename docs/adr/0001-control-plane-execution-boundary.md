# ADR 0001: Control-Plane Execution Boundary

- Status: Accepted
- Date: 2026-02-16
- Owners: proxmox-agent maintainers
- Related beads: `br-7ak`, `br-3sl`

## Context

This project must run safe Proxmox operations across home and cloud environments.
We need strong guardrails for state-changing actions and a clear split between orchestration and execution.

Without a strict boundary, intent interpretation and infrastructure mutation can mix in one component, increasing risk and reducing auditability.

## Decision

Use `pi agent` as the primary orchestration runtime and keep `proxmox-agent` as a constrained execution boundary.

`proxmox-agent` responsibilities:

- Validate incoming action requests.
- Enforce plan-first operation flow (`plan` before `apply`).
- Evaluate policy risk and approval requirements.
- Execute only explicit approved actions through the Proxmox client abstraction.
- Persist audit events for planning, denials, and applies.

`pi agent` responsibilities:

- Interpret user/operator intent.
- Select and sequence action requests.
- Collect required approvals for high-risk actions.
- Call `plan` and `apply` endpoints using the control-plane contract.

## Constraints and invariants

- High-risk actions require explicit approval metadata (`approved_by`) before apply.
- Home and cloud credentials remain isolated by environment.
- TLS verification must not be disabled in production environments.
- Token secrets are never logged or returned.

## Alternatives considered

1. Single autonomous service (orchestrates and executes): rejected due to weaker separation of duties and higher safety risk.
2. Direct Proxmox API calls from `pi agent` only: rejected because policy and audit guarantees become fragmented.

## Consequences

Positive:

- Stronger safety and compliance posture through policy and audit centralization.
- Clear ownership boundaries for orchestration vs execution logic.
- Easier extension of policy, approvals, and idempotency without changing agent behavior.

Tradeoffs:

- Requires stable API contracts between `pi agent` and `proxmox-agent`.
- Adds one service hop for every action.

## Follow-up

- Add canonical runtime contract documentation and examples.
- Add plan identifier and idempotency requirements.
- Add contract tests for `pi agent` integrations.
