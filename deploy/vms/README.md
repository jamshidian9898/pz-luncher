# VM Game-Server Lab — two Ubuntu VMs, QEMU/KVM + Ansible

Two real Ubuntu 24.04 server VMs, each running a **real Project Zomboid
Dedicated Server** (steamcmd, app 380870) with its own mod, plus a `pz-agent`
publishing to the platform backend on the host. Runs entirely inside WSL2
(or any Linux host) — no VirtualBox, no Windows-side setup.

```
host (WSL/Linux)                     VM pz-srv-1                VM pz-srv-2
┌──────────────┐   udp/16261  ┌─────────────────────┐   ┌─────────────────────┐
│ pz-backend   │◄─────────────│ PZ server (pzvm1)   │   │ PZ server (pzvm2)   │
│ :8080        │   HTTP       │ mods: LabModB       │   │ mods: LabModC       │
│              │◄─────────────│ pz-agent (systemd)  │   │ pz-agent (systemd)  │
└──────────────┘              └─────────────────────┘   └─────────────────────┘
```

## Prerequisites

- Linux/WSL2 with KVM (`/dev/kvm`, user in `kvm` group)
- `sudo apt-get install -y qemu-system-x86 qemu-utils cloud-image-utils ansible`
- Go 1.23+ (builds backend/agent from this repo)
- ~10 GB disk per VM, ~2.5 GB RAM per VM

## Usage

```bash
cd deploy/vms
./vms.sh up          # create or REUSE VMs + start host backend
./vms.sh provision   # Ansible: update, steamcmd, PZ server, mods, agent
./vms.sh status      # VMs, SSH, backend health, agent freshness
./vms.sh ssh 1       # shell into a VM
./vms.sh down        # graceful poweroff — disks kept
./vms.sh destroy     # delete disks (only way to lose VM state)
```

**Idempotent by design**: `up` reuses existing disks and running VMs (never
duplicates); `provision` converges — re-running it is always safe, and the
~4 GB PZ install is skipped once present. The steamcmd install continues
inside the VM even if the controller is interrupted; just re-run `provision`.

Options: `--root DIR` (default `~/pz-vms`), `--vms N`, `--backend-port P`,
`--mem MB`.

## Testing with the real game client

Each VM's PZ server is reachable from the host (and from Windows when the
host is WSL):

| Server | Connect to | Mods |
|---|---|---|
| pzvm1 | `localhost` port `16261` (UDP) | LabModB |
| pzvm2 | `localhost` port `16263` (UDP) | LabModC |

Admin credentials: `admin` / `pzlab-admin-1!`.

Platform side: backend at `http://localhost:8080` — the launcher (or
`go run ./apps/join-cli -server pzvm1 -backend http://localhost:8080`) joins,
downloads the server's mods, and builds a byte-identical profile.

To publish **real mods**, drop mod folders into `/home/pzserver/Zomboid/mods/`
inside a VM (`./vms.sh ssh 1`) and add their ids to `Mods=` in
`/home/pzserver/Zomboid/Server/pzvm1.ini`, then `sudo systemctl restart
pz-server`. The agent picks the content up on its next 60s sync.

## Notes

- VM SSH: `127.0.0.1:22001` / `22002`, user `pz`, key `~/pz-vms/keys/lab_ed25519`
- Logs: host `~/pz-vms/logs/`; in-VM `journalctl -u pz-server` / `-u pz-agent`
- The PZ JVM heap is capped at 1.5 GB to fit the VM (`pz_max_heap` in the playbook)
- After a WSL restart, just run `./vms.sh up` again — disks persist
