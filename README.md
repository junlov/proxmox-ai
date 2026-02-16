# proxmox-agent

A Go control-plane service for managing multiple Proxmox VE environments (home + cloud) with safety guardrails.

## Runtime model

- Any orchestration agent/runtime can call this service.
- `proxmox-agent` is the execution and policy boundary for Proxmox actions.
- State-changing operations must follow `plan` -> approval (if required) -> `apply`.
- High-risk apply requests must include explicit approver identity (`approved_by`).

## MVP scope

- Multi-environment config (`home`, `cloud`)
- Readiness and health endpoints
- Plan-first action pipeline (`plan` -> `apply`)
- Policy checks for destructive actions
- Audit log of all requested actions

## Quick start

```bash
go run ./cmd/proxmox-agent --config ./config.example.json
```

In another terminal:

```bash
export PROXMOX_AGENT_API_TOKEN='change-me'
curl -s localhost:8080/healthz | jq
curl -s -H "Authorization: Bearer $PROXMOX_AGENT_API_TOKEN" localhost:8080/v1/environments | jq
```

## Read-only inventory (VM + LXC)

Simple endpoint (plan + apply handled server-side):

```bash
curl -s \
  -H "Authorization: Bearer $PROXMOX_AGENT_API_TOKEN" \
  -H "X-Actor-ID: local-operator" \
  "localhost:8080/v1/inventory?environment=home&state=running" | jq
```

List all VM/LXC resources in an environment:

```bash
curl -s \
  -H "Authorization: Bearer $PROXMOX_AGENT_API_TOKEN" \
  -H "X-Actor-ID: local-operator" \
  -H "Content-Type: application/json" \
  -d '{"environment":"home","action":"read_inventory","target":"inventory/all"}' \
  localhost:8080/v1/actions/apply | jq
```

List only running VM/LXC resources:

```bash
curl -s \
  -H "Authorization: Bearer $PROXMOX_AGENT_API_TOKEN" \
  -H "X-Actor-ID: local-operator" \
  -H "Content-Type: application/json" \
  -d '{"environment":"home","action":"read_inventory","target":"inventory/running"}' \
  localhost:8080/v1/actions/apply | jq
```

## API (MVP)

- `GET /healthz`
- `GET /v1/environments`
- `GET /v1/inventory?environment=<name>&state=<all|running>`
- `POST /v1/actions/plan`
- `POST /v1/actions/apply`

Versioning and deprecation policy: `docs/api-versioning-policy.md`.

## Safety model

- Every request is validated and planned before execution.
- High-risk actions (delete, migrate, storage changes) require explicit approval.
- Dry-run mode is supported for all actions.
- Action requests are appended to `./data/audit.log`.

See `docs/runtime-contract.md` for the `pi agent` orchestration contract.

## Task tracking

This repo uses `br` (beads_rust):

```bash
br list
br ready
br show br-1
```
