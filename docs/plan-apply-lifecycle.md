# Plan/Apply Lifecycle State Machine

## Purpose

Define valid lifecycle states for action execution and the allowed transitions between them.

## States

- `draft`: request accepted for evaluation (transient/internal).
- `planned`: plan completed and decision issued.
- `denied`: policy denied apply eligibility.
- `awaiting_approval`: plan requires approval metadata before apply.
- `approved`: approval requirements satisfied for apply.
- `applying`: execution in progress.
- `applied`: execution completed successfully.
- `failed`: execution attempted and failed.
- `expired`: plan no longer valid due to TTL or explicit invalidation.
- `canceled`: execution canceled before completion.

## Terminal states

- `denied`
- `applied`
- `failed`
- `expired`
- `canceled`

## Valid transitions

- `draft -> planned`
- `planned -> denied`
- `planned -> awaiting_approval`
- `planned -> approved` (when no approval required)
- `awaiting_approval -> approved`
- `awaiting_approval -> expired`
- `approved -> applying`
- `approved -> expired`
- `applying -> applied`
- `applying -> failed`
- `applying -> canceled`

## Invalid transitions (examples)

- `denied -> applying`
- `expired -> approved`
- `applied -> applying`
- `failed -> applying` (must create a new plan)

## Error conditions

- Missing required fields (`environment`, `target`): reject request as invalid.
- Missing approval for required high-risk apply: transition to or remain in `awaiting_approval`/deny apply.
- Plan not found or expired at apply time: return apply error and state `expired`.
- Payload mismatch between plan and apply: return integrity error and no transition to `applying`.

## Notes for implementation

- The current MVP stores plan/apply audit events but does not yet persist full lifecycle state.
- Subsequent tasks should add persisted `plan_id`, status transitions, and replay protection.
