# Policy Risk Matrix

## Purpose

Define baseline risk, approval requirements, and policy outcomes for each action.

## Risk classes

- `low`: read-only or no service impact
- `medium`: state-changing but typically reversible and scoped
- `high`: destructive, security-impacting, or broad operational impact

## Mapping table (v1)

| Canonical action | Current internal action | Risk | Requires approval |
| --- | --- | --- | --- |
| `vm.read` | `read_vm` | low | no |
| `vm.start` | `start_vm` | medium | no |
| `vm.stop` | `stop_vm` | medium | yes |
| `vm.snapshot.create` | `snapshot_vm` | medium | no |
| `vm.migrate` | `migrate_vm` | high | yes |
| `vm.delete` | `delete_vm` | high | yes |
| `storage.edit` | `storage_edit` | high | yes |
| `firewall.edit` | `firewall_edit` | high | yes |

## Baseline policy outcomes

- If `requires_approval=true` and `approved_by` is missing, deny apply.
- If `environment` or `target` is missing, reject request as invalid.
- Plan evaluates risk and requirements even when apply is not allowed.

## Notes

- This matrix defines baseline behavior for MVP and should remain backward-compatible unless versioned.
- Future phases can add environment-specific overrides (for example stricter cloud rules).
