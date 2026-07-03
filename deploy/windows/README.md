# Windows Test Lab — single-machine proof of work

Everything on **one Windows OS**: backend, N Project Zomboid dedicated
servers, one agent per server, and a verifier that proves the full platform
loop with real files:

```
agent scans mods dir ──► deterministic-zip blobs + versioned manifest ──► backend
                                                                            │
      byte-identical profile ◄── extract ◄── download ◄── POST /join ◄── join-cli
```

## Prerequisites

- Windows 10/11 (or the repo's Vagrant Windows 11 box)
- Go 1.23+ on PATH (`choco install golang`)
- Elevated PowerShell (Administrator)
- ~6 GB disk for the PZ Dedicated Server (skippable)

## Quick start

```powershell
cd deploy\windows

.\lab.ps1 install     # build binaries, seed mods, install PZ server + services
.\lab.ps1 up          # start backend + agents (+ game servers)
.\lab.ps1 verify      # prove the loop; exit code 0 = PASS
.\lab.ps1 status      # services, game servers, agent freshness
.\lab.ps1 down        # stop everything
.\lab.ps1 clean       # down + uninstall the Windows services
```

Fast platform-only lab (no steamcmd, no 4 GB download — game servers skipped,
platform loop still fully proven):

```powershell
.\lab.ps1 install -SkipGameServers
.\lab.ps1 up -SkipGameServers
.\lab.ps1 verify
```

## What `install` sets up

| Component | How | Where |
|---|---|---|
| `pz-backend.exe` | native Windows service `PZBackend`, auto-start, restart-on-failure | `C:\pz-lab\bin` |
| `pz-agent.exe` ×N | services `PZAgent1..N`, one per game server, 60s sync | `C:\pz-lab\bin` |
| PZ Dedicated Server | steamcmd, app 380870, shared install | `C:\pz-lab\pzserver` |
| Server instances | `-cachedir C:\pz-lab\data\srvN`, ports 16261/16263/… | `C:\pz-lab\data\srvN` |
| Sample mods | seeded per server (only if mods dir empty) | `C:\pz-lab\data\srvN\mods` |
| Logs | backend, agents, game servers, verify | `C:\pz-lab\logs` |

Defaults are overridable: `-Root D:\lab -Servers 3 -BackendPort 9090`.

To publish **real** mods instead of the seeded samples, drop mod folders into
`C:\pz-lab\data\srvN\mods` — the agent picks them up on the next 60s sync and
`verify` will check byte-for-byte fidelity against the extracted profile.

## What `verify` asserts

1. Backend `/api/v1/health` is `ok`
2. Every `pzlabN` server was auto-registered by its agent
3. Every agent is `online` (fresh heartbeat)
4. A versioned manifest exists for every server
5. `join-cli -backend` (the same `POST /join` + v2 pipeline the launcher UI
   uses) completes against the live backend
6. The extracted profile under `C:\pz-lab\launcher\profiles\pzlab1\mods` is
   **byte-identical** to the server's mods directory

Non-zero exit on any failure — safe to call from CI or a scheduled task.

## Using the Vagrant Windows 11 box

From the repo root on the host:

```bash
vagrant up
vagrant provision --provision-with lab   # runs lab.ps1 install+up (platform-only)
vagrant powershell -c "C:\Users\vagrant\project\deploy\windows\lab.ps1 verify"
```

For the full game-server lab inside the VM, give it more resources in the
`Vagrantfile` (8 GB+ RAM) and run `lab.ps1 install` without `-SkipGameServers`.
