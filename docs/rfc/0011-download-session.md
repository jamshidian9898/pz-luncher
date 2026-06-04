# RFC 0011: Download Session

## Problem

Downloads need stateful tracking for resume, progress, retries, and failure handling.

## Goals

- Capture download session state for each package and profile
- Support resume and retry semantics
- Expose progress and error details to the launcher
- Keep session state consistent and recoverable

## Session model

Fields
- `id` (string)
- `profileId` (string)
- `packageHash` (string)
- `provider` (string)
- `state` (`pending`, `in_progress`, `completed`, `failed`)
- `progress` (float64)
- `startedAt` (timestamp)
- `lastUpdatedAt` (timestamp)
- `error` (string)

## Download state machine

- `pending` → `in_progress`
- `in_progress` → `completed`
- `in_progress` → `failed`
- `failed` → `pending` when retrying

## Behavior

- incomplete downloads are resumed from the last known state
- download sessions persist at least until completion or cancellation
- progress updates are throttled for UI and service efficiency
- failed sessions include reason metadata for diagnostics

## Open Questions

- Should session retries be automatic or manual?
- How long should failed sessions be retained?
- Should sessions be grouped by profile or package batch?
