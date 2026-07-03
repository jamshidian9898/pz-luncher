# Linux Test Lab — Ubuntu / WSL, single machine

Same proof-of-work lab as `deploy/windows`, native on Linux. Everything on
**one machine**: backend, N Project Zomboid dedicated servers, one agent per
server, and a verifier that proves the full platform loop with real files:

```
agent scans mods dir ──► deterministic-zip blobs + versioned manifest ──► backend
                                                                            │
   byte-identical profile ◄── extract ◄── download ◄── POST /join ◄── join-cli
                     │
                     └──► game launched with -cachedir=<profile>
```

## Prerequisites

- Ubuntu (server or WSL) with Go 1.23+ on PATH
- `python3` and `curl` (present on stock Ubuntu)
- For full game servers: sudo (installs `lib32gcc-s1` for steamcmd), ~6 GB disk

## Quick start

```bash
cd deploy/linux

./lab.sh install --skip-game-servers   # fast platform-only lab
./lab.sh up --skip-game-servers
./lab.sh verify                        # exit 0 = PASS
./lab.sh status
./lab.sh down
./lab.sh clean
```

Full lab with real PZ dedicated servers (steamcmd, app 380870):

```bash
./lab.sh install
./lab.sh up
./lab.sh verify
```

Options on every command: `--root DIR` (default `~/pz-lab`), `--servers N`
(default 2), `--backend-port P` (default 8080), `--no-systemd`.

## Service management

Detected automatically:

| Environment | Mode |
|---|---|
| Ubuntu server / WSL2 with `systemd=true` | systemd units `pz-backend`, `pz-agent1..N` (enabled at boot, restart-on-failure) |
| Plain WSL (no systemd) | pid-file background processes under `$ROOT/run/` |

Force pid-file mode with `--no-systemd`. Game servers always run as tracked
background processes (`$ROOT/run/srvN.pid`), logs in `$ROOT/logs/`.

## What `verify` asserts

1. Backend `/api/v1/health` is `ok`
2. Every `pzlabN` server was auto-registered by its agent
3. Every agent is `online` (fresh heartbeat)
4. A versioned manifest exists for every server
5. `join-cli -backend` (the same `POST /join` + v2 pipeline the launcher UI
   uses) completes against the live backend
6. The launch flow starts the game with `-cachedir=<profile>` (asserted via a
   fake `ProjectZomboid64` binary that records its argv)
7. The extracted profile under `$ROOT/launcher/profiles/pzlab1/mods` is
   **byte-identical** to the server's mods directory

Non-zero exit on any failure — safe for CI or cron.

To publish **real** mods instead of the seeded samples, drop mod folders into
`$ROOT/data/srvN/mods` — the agent picks them up on its next 60s sync.
