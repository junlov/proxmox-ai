# Control-Plane Expansion Roadmap

## Goal

Expand proxmox-agent from MVP to a full control-plane for provisioning, storage, backup, DR, network, and observability while preserving plan/apply safety and audit guarantees.

## Principles

- All state-changing actions require plan then apply.
- High-risk actions require explicit approval metadata.
- Home and cloud credentials stay isolated by environment.
- TLS verification stays enabled in production.
- Token secrets never appear in logs or audit payloads.
- Each feature ships with action schemas, policy mapping, and tests.

## Prerequisites (blocking)

- `br-ya9`: Wire real Proxmox API client
- `br-1sa`: Add authentication for API endpoints
- `br-2vy`: Add idempotency key support
- `br-hpx`: Implement policy gates for risky actions
- `br-124`: Add environment credential loading

## Feature Done Criteria

- Action schema added to the canonical taxonomy or v2 schema set.
- Policy risk class and approval requirements defined.
- Plan/apply path wired and audited.
- Error responses are structured and redact secrets.
- Tests cover validation and runner behavior.
- Runbook or usage docs included where applicable.

## Delivery Phases

### Phase 0: Foundation v2 (br-u6b.1)

Scope:
- Define action schema v2 for provisioning, storage, backup, and network.
- Approval policy v2 (risk classes, change windows, approver metadata).
- Audit redaction and secret-safe parameter logging.
- Integration test harness for async Proxmox task workflows.

### Phase 1: Provisioning Automation (br-u6b.2)

Scope:
- `create_vm_from_template` action (plan/apply).
- Cloud-init configuration action.
- Post-provision bootstrap action (guest agent, SSH profile).
- `/v1/vm/status` read endpoint.
- Provisioning runbook and API examples.

### Phase 2: Storage Lifecycle Automation (br-u6b.3)

Scope:
- Attach and detach disk actions.
- Resize disk action with preflight checks.
- Move disk action with rollback guidance.
- Capacity guardrails and saturation alarms.

### Phase 3: Snapshot Policy Automation (br-u6b.5)

Scope:
- Snapshot schedule policy model per VM class.
- Snapshot executor with lock-aware task waits.
- Snapshot TTL prune and safety constraints.

### Phase 4: Backup Automation (br-u6b.4)

Scope:
- Backup job CRUD actions.
- Run-backup-now action with task tracking.
- Retention policy engine and automated prune.
- Restore validation workflow.

### Phase 5: DR and Migration Workflows (br-u6b.6)

Scope:
- Migration precheck action (HA state, storage/network compatibility).
- Controlled migrate workflow with rollback checkpoints.
- Replication policy actions for cross-environment DR.

### Phase 6: Network and Firewall Policy (br-u6b.7)

Scope:
- Network profile templates (bridge, VLAN, IP plan).
- Firewall template actions with high-risk approval enforcement.
- Drift detection and reconciliation plan.

### Phase 7: Observability (br-u6b.8)

Scope:
- Event stream endpoint for task and policy decisions.
- Health summary APIs.
- Webhook alerts for failures and policy violations.

### Phase 8: Curated Host Actions (br-u6b.10)

Scope:
- Allowlisted SSH host-action runner with strict command policy.
- Curated host actions like `install_umbrel_vm`.
- Maintenance actions (apt updates, reboot windows).

### Phase 9: Intent and ChatOps (br-u6b.9)

Scope:
- Intent endpoints such as `ensure_vm_running` and `ensure_backup_policy`.
- ChatOps adapters (GitHub issue comments and Slack webhook).

## Tracking

Execution work is tracked in the `br-u6b.*` child issues. This document defines the high-level sequencing and scope boundaries.

## Notes

This roadmap aligns with the canonical action taxonomy (`docs/action-taxonomy.md`) and policy risk matrix (`docs/policy-risk-matrix.md`). Each phase should preserve the plan/apply lifecycle and audit requirements described in `docs/plan-apply-lifecycle.md` and `docs/runtime-contract.md`.
