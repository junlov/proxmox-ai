# Canonical Action Taxonomy

## Purpose

Define stable action names and semantics shared by `pi agent`, API clients, and `proxmox-agent`.

## Naming rules

- Use dotted verbs for canonical names: `<domain>.<operation>`.
- Keep names explicit and side-effect oriented.
- Avoid aliases in API payloads; aliases may exist only in client translation layers.

## Canonical actions (v1)

- `vm.read`
- `vm.start`
- `vm.stop`
- `vm.snapshot.create`
- `vm.migrate`
- `vm.delete`
- `storage.edit`
- `firewall.edit`

## Risk mapping baseline

- Low: `vm.read`
- Medium: `vm.start`, `vm.stop`, `vm.snapshot.create`
- High: `vm.migrate`, `vm.delete`, `storage.edit`, `firewall.edit`

High-risk actions require explicit approval metadata before apply.

## Legacy mapping (current internal constants)

- `read_vm` -> `vm.read`
- `start_vm` -> `vm.start`
- `stop_vm` -> `vm.stop`
- `snapshot_vm` -> `vm.snapshot.create`
- `migrate_vm` -> `vm.migrate`
- `delete_vm` -> `vm.delete`
- `storage_edit` -> `storage.edit`
- `firewall_edit` -> `firewall.edit`

## Semantics

- All state-changing actions must run via `plan` then `apply`.
- `dry_run=true` means no mutation, but full validation and policy evaluation still apply.
- Action `target` must resolve to a concrete object (no wildcard destructive operations).
