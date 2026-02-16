# API Versioning and Deprecation Policy

## Scope

Applies to `proxmox-agent` HTTP APIs and runtime contract documents consumed by `pi agent`.

## Versioning model

- Major API versions use path prefixes (`/v1`, `/v2`).
- Backward-compatible changes are allowed within a major version.
- Breaking changes require a new major version.

## Backward-compatible changes (allowed in v1)

- Additive response fields.
- New optional request fields.
- New endpoints that do not alter existing semantics.
- Expanded enum values only when clients can safely ignore unknown values.

## Breaking changes (require new major version)

- Removing or renaming fields.
- Changing required fields or validation semantics incompatibly.
- Changing risk/approval behavior in a way that weakens safety guarantees.
- Changing endpoint paths or auth model incompatibly.

## Deprecation policy

- Mark feature/field as deprecated in docs and release notes.
- Provide migration guidance and replacement behavior.
- Keep deprecated behavior available for at least one minor release cycle before removal.
- Emit warning metadata where practical for deprecated usage.

## Contract governance

- Runtime contract docs (`runtime-contract`, `action-taxonomy`, policy docs) are source-of-truth artifacts.
- Contract-affecting code changes must update docs in the same change set.
