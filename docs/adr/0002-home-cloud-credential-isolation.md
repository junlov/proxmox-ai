# ADR 0002: Home and Cloud Credential Isolation

- Status: Accepted
- Date: 2026-02-16
- Owners: proxmox-agent maintainers
- Related beads: `br-3p2`

## Context

The control-plane manages multiple Proxmox environments (`home`, `cloud`).
Credential reuse across these environments increases blast radius and can turn a single compromise into a full-fleet incident.

## Decision

Enforce strict home/cloud credential isolation in configuration, runtime behavior, and operational process.

## Rules

- Each environment must use distinct token IDs and token secrets.
- Secrets are referenced by environment-specific env vars (`token_secret_env`) and never persisted in repo config.
- Tokens must be least-privilege and scoped to required operations only.
- Automation roles are preferred over full admin principals.
- Requests for one environment must never read or use credentials from another environment.

## Operational constraints

- Production environments must keep TLS certificate verification enabled.
- Token secrets must not be logged, echoed, or returned through API responses.
- Secret rotation should be performed per environment, independently.

## Alternatives considered

1. Shared cross-environment automation token: rejected due to excessive blast radius.
2. Single root/admin token per environment: rejected as default because it violates least-privilege goals.

## Consequences

Positive:

- Reduced impact from credential leakage.
- Clear ownership and rotation boundaries by environment.
- Better policy and audit clarity per environment.

Tradeoffs:

- Increased credential management overhead.
- Additional setup complexity for operators.

## Follow-up

- Add startup validation for duplicate token IDs across environments.
- Add policy checks that enforce production TLS constraints.
- Add runbook for independent home/cloud secret rotation.
