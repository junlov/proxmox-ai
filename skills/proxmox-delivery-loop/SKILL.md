---
name: proxmox-delivery-loop
description: Use when implementing proxmox-agent backlog items that must be delivered in small slices with strict safety, local tests, live Proxmox validation, and GitHub push after each completed task.
---

# Proxmox Delivery Loop

## Overview

This skill standardizes how to execute one backlog task at a time in `proxmox-agent` while keeping safety controls (`plan -> apply`, TLS verification, no secret logging) and a tight delivery loop (code, test, live validate, commit, push).

Use this for feature work that touches Proxmox actions/endpoints and must be proven against a real Proxmox node before task closure.

## Workflow

1. Pick one task
- Identify a single `br` issue to complete.
- Keep scope to one deployable increment.

2. Implement minimally
- Prefer read-only actions/endpoints first.
- For state-changing actions, enforce existing policy gates.
- Avoid broad refactors unless required by the task.

3. Validate locally
- Run:
```bash
GOCACHE=$(pwd)/.tmp/gocache gofmt -w ./cmd ./internal
GOCACHE=$(pwd)/.tmp/gocache go test ./...
```

4. Validate live on Proxmox
- Required env vars:
```bash
export SSL_CERT_FILE="$(pwd)/.tmp/certs/pve-root-ca.pem"
export PROXMOX_AGENT_API_TOKEN='<agent-api-token>'
export PVE_PVE_TOKEN_SECRET='<proxmox-token-secret>'
```
- Start service, run a focused endpoint/action check, capture response, stop service.
- Record HTTP status and key fields proving the task works.

5. Commit and push immediately
- One commit per completed task.
- Push to `origin/main` right after commit.

6. Close the task
- Close the `br` task only after live validation succeeds.
- Include close reason: `Implemented with tests and live Proxmox validation`.

## Safety Rules

- Never disable TLS verification.
- Never store token secrets in repo files.
- Do not log token secrets in command output summaries.
- Respect plan/apply lifecycle for mutations.
- High-risk actions require explicit approval metadata.

## Command Templates

Create/update one task:
```bash
br create -t task -p P1 --parent <epic-id> "<task title>"
```

Run one live validation:
```bash
(go run ./cmd/proxmox-agent --config ./config.example.json > .tmp/run/agent.log 2>&1 &) \
&& pid=$! && sleep 2 \
&& curl -sS -H "Authorization: Bearer $PROXMOX_AGENT_API_TOKEN" \
   -H "X-Actor-ID: local-operator" \
   "http://127.0.0.1:8080/<endpoint>" \
&& kill "$pid" && wait "$pid" 2>/dev/null || true
```

Commit/push/close:
```bash
br sync --flush-only
git add <files>
git commit --no-verify -m "<task commit message>"
git push origin main
br close <task-id> --force --reason "Implemented with tests and live Proxmox validation"
```

## Done Criteria Per Task

- Code is formatted and tests pass.
- Endpoint/action validated against real Proxmox data.
- Commit pushed to GitHub.
- `br` task closed with validation reason.
