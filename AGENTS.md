# AGENTS.md

## Objective
Build and operate a safe Proxmox control-plane for home and cloud environments.

## Safety rules
- Default to read-only behavior unless explicitly asked to change state.
- Always run `plan` before `apply`.
- Never execute high-risk actions without explicit approval (`approved_by` required).
- High-risk actions include: VM delete, VM migrate, storage edits, firewall edits.
- Do not disable TLS verification in production environments.
- Never print token secrets or write them to logs.

## Environment policy
- Keep home and cloud credentials separate.
- Use least-privilege tokens per environment.
- Prefer dedicated automation roles over full admin roles.

## Development workflow
- Keep all work in this repo.
- Run before finishing changes:
  - `gofmt -w ./cmd ./internal`
  - `go test ./...`

## Task tracking (`br`)
- Create work items with `br create`.
- Link blocking work with `br dep add <issue> <depends-on>`.
- Check ready queue with `br ready`.
- Export JSONL before commit/push:
  - `br sync --flush-only`
  - `git add .beads/`

## Commit guidance
- Keep commits small and scoped.
- Mention relevant issue IDs in commit messages when possible.

## Local skills
- `proxmox-delivery-loop`: `skills/proxmox-delivery-loop/SKILL.md`
  - Use when implementing backlog items that must be completed one-by-one with:
    - local `gofmt` + `go test`
    - live Proxmox validation
    - immediate commit/push after each completed task

<!-- bv-agent-instructions-v1 -->

---

## Beads Workflow Integration

This project uses [beads_viewer](https://github.com/Dicklesworthstone/beads_viewer) for issue tracking. Issues are stored in `.beads/` and tracked in git.

### Essential Commands

```bash
# View issues (launches TUI - avoid in automated sessions)
bv

# CLI commands for agents (use these instead)
bd ready              # Show issues ready to work (no blockers)
bd list --status=open # All open issues
bd show <id>          # Full issue details with dependencies
bd create --title="..." --type=task --priority=2
bd update <id> --status=in_progress
bd close <id> --reason="Completed"
bd close <id1> <id2>  # Close multiple issues at once
bd sync               # Commit and push changes
```

### Workflow Pattern

1. **Start**: Run `bd ready` to find actionable work
2. **Claim**: Use `bd update <id> --status=in_progress`
3. **Work**: Implement the task
4. **Complete**: Use `bd close <id>`
5. **Sync**: Always run `bd sync` at session end

### Key Concepts

- **Dependencies**: Issues can block other issues. `bd ready` shows only unblocked work.
- **Priority**: P0=critical, P1=high, P2=medium, P3=low, P4=backlog (use numbers, not words)
- **Types**: task, bug, feature, epic, question, docs
- **Blocking**: `bd dep add <issue> <depends-on>` to add dependencies

### Session Protocol

**Before ending any session, run this checklist:**

```bash
git status              # Check what changed
git add <files>         # Stage code changes
bd sync                 # Commit beads changes
git commit -m "..."     # Commit code
bd sync                 # Commit any new beads changes
git push                # Push to remote
```

### Best Practices

- Check `bd ready` at session start to find available work
- Update status as you work (in_progress → closed)
- Create new issues with `bd create` when you discover tasks
- Use descriptive titles and set appropriate priority/type
- Always `bd sync` before ending session

<!-- end-bv-agent-instructions -->
