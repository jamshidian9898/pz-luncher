# Domain: Download Session

## Entity: Download Session

Fields
- `id` (string): unique download session identifier
- `profileId` (string): profile initiating the download
- `packageHash` (string): targeted package hash
- `provider` (string): provider used for the download
- `state` (string): `pending`, `in_progress`, `completed`, `failed`
- `progress` (float64): completion percentage
- `startedAt` (timestamp)
- `lastUpdatedAt` (timestamp)
- `error` (string): optional failure reason

Rules
- session ids are unique per download request
- `packageHash` identifies the package content being downloaded
- `state` transitions must be monotonic and valid
- `progress` is 0–100
- `provider` must be selected from available providers

State transitions
- pending → in_progress when the download begins
- in_progress → completed when all chunks finish and integrity passes
- in_progress → failed if download or verification fails
- failed → pending when retrying

Relations
- Download sessions belong to a profile and a package
- Download manager holds and updates session state
- Sessions may be retried, resumed, or cancelled
