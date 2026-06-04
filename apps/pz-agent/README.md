# pz-agent

This folder contains the server-side agent foundation.

It defines the `Agent` interface for:

- `Heartbeat()`
- `GenerateManifest()`
- `DetectMods()`
- `Sync()`

The agent is intentionally kept independent from backend services for the foundation layer.
