# RFC 0016: Game Launcher

## Problem

The launcher must find the installed Project Zomboid client and start it with the correct profile and launch arguments.

## Goals

- Locate the Project Zomboid installation path
- Resolve Steam or native launch URIs
- Build correct launch arguments for the selected profile
- Support compatibility across Windows, macOS, and Linux

## Responsibility

- find the game executable or Steam app URI
- construct arguments for profile path, server config, and mod load order
- validate the target game version against the manifest
- handle Steam launch behavior and fallback to direct executable launch

## Launch process

1. discover installed Project Zomboid path
2. validate the installation version
3. build launch arguments for the selected profile
4. execute the game process or Steam URI
5. monitor process start and report status

## Invariants

- the game must not be launched without a prepared profile
- launch arguments must match the target game version and server requirements
- unsupported launch paths should fail early with a clear error
