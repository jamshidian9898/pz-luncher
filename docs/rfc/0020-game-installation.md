# RFC 0020: Game Installation

## Problem

The launcher must discover the installed Project Zomboid client and derive launch metadata without relying on backend services.

## Goals

- locate the installed Project Zomboid executable or Steam URI
- detect installed build/version metadata
- build launch arguments for profiles and servers
- support cross-platform detection for Windows, macOS, and Linux

## Responsibilities

- `GameInstallation` should represent discovered installation details
- discover installation paths and Steam integration metadata
- validate that the discovered build matches manifest requirements
- expose launch arguments and executable path for launchers

## Invariants

- game launch must only proceed if a valid installation is found
- version detection should be explicit and auditable
- Steam and direct launch paths should be handled separately
