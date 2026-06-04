# Contract: Download Service

## Responsibility

The download service retrieves package content reliably, with support for resume, chunks, and integrity verification.

## API surface

- `GET /packages/{hash}/download`
  - returns the package blob as a stream
- `GET /download/status/{sessionId}`
  - returns download session state and progress
- `POST /download/retry/{sessionId}`
  - retry a failed download session

## Data contracts

- downloads reference `packageHash`, `provider`, `destination`, and session metadata
- sessions expose `state`, `progress`, `error`, and timestamps

## Invariants

- downloads must validate the blob hash before marking complete
- partial downloads must be resumable by session id or chunk range
- failed downloads retain diagnostics for retry or rollback

## Integration points

- launcher core drives download sessions for missing packages
- registry service provides package metadata and download URLs
- provider system may be used to source download endpoints
