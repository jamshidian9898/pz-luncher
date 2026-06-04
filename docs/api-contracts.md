# API Contracts

These API contracts define the interactions between clients, services, and agents.

## Directory Service

- `GET /servers`
  - returns public servers with metadata, tags, player counts, and manifest references

- `GET /servers/{id}`
  - returns server details, supported packs, and direct join metadata

- `POST /heartbeat`
  - server-side heartbeat updates server status and availability

## Manifest Service

- `GET /manifest/{server}`
  - returns the latest manifest for the server

- `GET /manifest/{server}/{version}`
  - returns a specific manifest version

- `POST /manifest/{server}`
  - upload or update a server manifest

- `POST /rollback`
  - request rollback to an earlier manifest version

## Registry Service

- `POST /packages`
  - publish a package by hash and metadata

- `GET /packages/{hash}`
  - retrieve package metadata

- `GET /packages/{hash}/download`
  - download package content

## Agent API

- `POST /heartbeat`
  - agent sends health and status information

- `PUT /manifest`
  - agent submits generated manifest data
