# Architecture (MVP)

Reference ADR: `docs/adr/0001-control-plane-execution-boundary.md`
Action naming reference: `docs/action-taxonomy.md`
Policy risk reference: `docs/policy-risk-matrix.md`
Lifecycle reference: `docs/plan-apply-lifecycle.md`
Plan ID reference: `docs/plan-id-rules.md`
Request hashing reference: `docs/request-hashing.md`
Idempotency reference: `docs/idempotency.md`
Versioning policy reference: `docs/api-versioning-policy.md`

## Components

- `cmd/proxmox-agent`: process entrypoint
- `internal/server`: HTTP API for plan/apply flows
- `internal/policy`: policy engine (risk + approval gates)
- `internal/proxmox`: Proxmox client abstraction
- `internal/actions`: orchestrates plan/apply and audit trail

## Control-plane boundary

- `pi agent` is the primary agent runtime and orchestrator.
- `proxmox-agent` is intentionally narrow: it validates requests, evaluates policy, audits, and executes approved actions.
- The service should not infer autonomous intent; it only executes explicit action requests.
- Write actions must be modeled as plan-first workflows (`plan` before `apply`).

## Runtime contract (pi agent -> proxmox-agent)

1. `pi agent` submits `POST /v1/actions/plan` with a normalized action request.
2. `proxmox-agent` evaluates policy and returns risk, approval requirements, and decision.
3. `pi agent` obtains human/automation approval for high-risk actions.
4. `pi agent` submits `POST /v1/actions/apply` with approval metadata where required.
5. `proxmox-agent` re-evaluates policy, executes the action, and records immutable audit evidence.

## Execution model

1. Agent/UI submits action request to `/v1/actions/plan`
2. Policy engine returns risk and approval requirements
3. If approved, client submits `/v1/actions/apply`
4. Runner records audit event and executes action

## Next milestones

- Replace stub client with real Proxmox API client (`/api2/json`)
- Add idempotency keys and retry semantics
- Add approval workflow integration (chat/CLI)
- Add per-environment RBAC and token scopes
- Add tests and CI
