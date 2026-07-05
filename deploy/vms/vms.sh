#!/usr/bin/env bash
#
# PZ Platform — two-VM game-server lab (QEMU/KVM inside WSL2 or any Linux host).
#
# Creates N Ubuntu 24.04 server VMs (cloud image + cloud-init), provisions them
# with Ansible (system update, steamcmd, real PZ Dedicated Server, per-server
# mods, pz-agent as systemd service), and runs the platform backend on the host.
#
# Idempotent: existing VM disks are REUSED on the next run (never recreated,
# never duplicated); a running VM is left alone; Ansible converges state.
#
# Usage:
#   ./vms.sh up          # create/boot VMs (reuses existing), start host backend
#   ./vms.sh provision   # run Ansible against the VMs (update + install + mods)
#   ./vms.sh status      # VMs, SSH, backend, agents
#   ./vms.sh ssh 1       # shell into VM 1
#   ./vms.sh down        # graceful poweroff (disks kept)
#   ./vms.sh destroy     # down + DELETE disks (fresh start next up)
#
# Options: --root DIR (default ~/pz-vms)  --vms N (default 2)
#          --backend-port P (default 8080)  --mem MB (default 2560)
set -euo pipefail

COMMAND=${1:-status}
[ $# -gt 0 ] && shift

ROOT="${PZ_VMS_ROOT:-$HOME/pz-vms}"
VMS=2
BACKEND_PORT=8080
MEM=2560
SSH_ARG=""

while [ $# -gt 0 ]; do
  case "$1" in
    --root) ROOT="$2"; shift 2 ;;
    --vms) VMS="$2"; shift 2 ;;
    --backend-port) BACKEND_PORT="$2"; shift 2 ;;
    --mem) MEM="$2"; shift 2 ;;
    [0-9]*) SSH_ARG="$1"; shift ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKEND_URL="http://localhost:$BACKEND_PORT"
IMG_URL="https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
BASE_IMG="$ROOT/images/noble-server-cloudimg-amd64.img"

log()  { printf '\033[36m[vms]\033[0m %s\n' "$*"; }
ok()   { printf '  \033[32mPASS\033[0m  %s\n' "$*"; }
fail() { printf '  \033[31mFAIL\033[0m  %s\n' "$*"; }

vm_name()     { echo "pz-srv-$1"; }
vm_disk()     { echo "$ROOT/disks/pz-srv-$1.qcow2"; }
vm_seed()     { echo "$ROOT/seed/pz-srv-$1.iso"; }
vm_pidfile()  { echo "$ROOT/run/pz-srv-$1.pid"; }
vm_ssh_port() { echo $((22000 + $1)); }
# Host UDP ports forwarded to the PZ server inside each VM (16261/16262).
vm_game_port()   { echo $((16261 + ($1 - 1) * 2)); }
vm_direct_port() { echo $((16262 + ($1 - 1) * 2)); }
server_name()    { echo "pzvm$1"; }
seed_mod()       { echo "LabMod$(tr '0-9' 'ABCDEFGHIJ' <<<"$1")"; }

SSH_KEY="$ROOT/keys/lab_ed25519"
SSH_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=3 -i "$SSH_KEY")

require_tools() {
  local missing=()
  for c in qemu-system-x86_64 qemu-img cloud-localds ansible-playbook curl go; do
    command -v "$c" >/dev/null || missing+=("$c")
  done
  if [ ${#missing[@]} -gt 0 ]; then
    echo "missing tools: ${missing[*]}" >&2
    echo "install with:" >&2
    echo "  sudo apt-get update && sudo apt-get install -y qemu-system-x86 qemu-utils cloud-image-utils ansible" >&2
    exit 1
  fi
  [ -c /dev/kvm ] || { echo "/dev/kvm not available — enable nested virtualization" >&2; exit 1; }
  [ -w /dev/kvm ] || { echo "/dev/kvm not writable — run: sudo usermod -aG kvm $USER (then restart the shell/WSL)" >&2; exit 1; }
}

vm_running() {
  local pidfile; pidfile="$(vm_pidfile "$1")"
  [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null
}

ensure_key() {
  [ -f "$SSH_KEY" ] && return
  mkdir -p "$ROOT/keys"
  ssh-keygen -t ed25519 -N '' -C pz-vms-lab -f "$SSH_KEY" >/dev/null
  log "generated lab SSH key $SSH_KEY"
}

ensure_base_image() {
  [ -f "$BASE_IMG" ] && return
  mkdir -p "$(dirname "$BASE_IMG")"
  log "downloading Ubuntu 24.04 cloud image (~600 MB, one time)"
  curl -fL --progress-bar -o "$BASE_IMG.part" "$IMG_URL"
  mv "$BASE_IMG.part" "$BASE_IMG"
}

ensure_seed() {
  local i=$1 seed; seed="$(vm_seed "$i")"
  [ -f "$seed" ] && return
  mkdir -p "$ROOT/seed"
  local userdata="$ROOT/seed/user-data-$i" metadata="$ROOT/seed/meta-data-$i"
  cat > "$userdata" <<EOF
#cloud-config
hostname: $(vm_name "$i")
users:
  - name: pz
    sudo: 'ALL=(ALL) NOPASSWD:ALL'
    shell: /bin/bash
    ssh_authorized_keys:
      - $(cat "$SSH_KEY.pub")
ssh_pwauth: false
EOF
  printf 'instance-id: %s\nlocal-hostname: %s\n' "$(vm_name "$i")" "$(vm_name "$i")" > "$metadata"
  cloud-localds "$seed" "$userdata" "$metadata"
}

start_vm() {
  local i=$1
  if vm_running "$i"; then
    log "$(vm_name "$i") already running (pid $(cat "$(vm_pidfile "$i")")) — reusing"
    return
  fi
  mkdir -p "$ROOT/disks" "$ROOT/run" "$ROOT/logs"
  local disk; disk="$(vm_disk "$i")"
  if [ -f "$disk" ]; then
    log "$(vm_name "$i") disk exists — reusing (no duplicate created)"
  else
    log "creating disk for $(vm_name "$i") (20G overlay on base image)"
    qemu-img create -q -f qcow2 -b "$BASE_IMG" -F qcow2 "$disk" 20G
  fi
  ensure_seed "$i"
  log "booting $(vm_name "$i")  ssh=127.0.0.1:$(vm_ssh_port "$i")  game=udp/$(vm_game_port "$i")"
  qemu-system-x86_64 \
    -name "$(vm_name "$i")" \
    -enable-kvm -cpu host -smp 2 -m "$MEM" \
    -drive "file=$disk,if=virtio,format=qcow2" \
    -drive "file=$(vm_seed "$i"),if=virtio,format=raw,readonly=on" \
    -netdev "user,id=n0,hostfwd=tcp:127.0.0.1:$(vm_ssh_port "$i")-:22,hostfwd=udp::$(vm_game_port "$i")-:16261,hostfwd=udp::$(vm_direct_port "$i")-:16262" \
    -device virtio-net-pci,netdev=n0 \
    -display none \
    -serial "file:$ROOT/logs/$(vm_name "$i")-console.log" \
    -daemonize -pidfile "$(vm_pidfile "$i")"
}

wait_ssh() {
  local i=$1 tries=60
  log "waiting for SSH on $(vm_name "$i") (first boot takes a minute)"
  while [ $tries -gt 0 ]; do
    if ssh "${SSH_OPTS[@]}" -p "$(vm_ssh_port "$i")" pz@127.0.0.1 true 2>/dev/null; then
      log "$(vm_name "$i") SSH is up"
      return 0
    fi
    sleep 5; tries=$((tries - 1))
  done
  echo "$(vm_name "$i") did not become reachable — see $ROOT/logs/$(vm_name "$i")-console.log" >&2
  return 1
}

write_inventory() {
  local inv="$ROOT/inventory.ini" i
  {
    echo "[pz_vms]"
    for i in $(seq 1 "$VMS"); do
      echo "$(vm_name "$i") ansible_host=127.0.0.1 ansible_port=$(vm_ssh_port "$i") pz_server_name=$(server_name "$i") seed_mod_id=$(seed_mod "$i")"
    done
    echo
    echo "[pz_vms:vars]"
    echo "ansible_user=pz"
    echo "ansible_ssh_private_key_file=$SSH_KEY"
    echo "ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'"
    echo "agent_local_bin=$ROOT/build/pz-agent"
    echo "pz_backend_url=http://10.0.2.2:$BACKEND_PORT"
  } > "$inv"
  log "wrote $inv"
}

ensure_backend() {
  local pidfile="$ROOT/run/backend.pid"
  if [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
    log "host backend already running (pid $(cat "$pidfile"))"
    return
  fi
  mkdir -p "$ROOT/backend/store" "$ROOT/logs" "$ROOT/run"
  [ -f "$ROOT/backend/registry.json" ] || echo '{"servers":[]}' > "$ROOT/backend/registry.json"
  log "building + starting host backend on :$BACKEND_PORT"
  (cd "$REPO_ROOT" && go build -o "$ROOT/bin/pz-backend" ./apps/backend/cmd/backend)
  setsid nohup "$ROOT/bin/pz-backend" \
    -addr ":$BACKEND_PORT" \
    -registry "$ROOT/backend/registry.json" \
    -store "$ROOT/backend/store" \
    -content-registry "$ROOT/backend/content-registry.json" \
    -fixtures "$ROOT/backend/fixtures" \
    >> "$ROOT/logs/pz-backend.log" 2>&1 &
  echo $! > "$pidfile"
}

cmd_up() {
  require_tools
  ensure_key
  ensure_base_image
  ensure_backend
  local i
  for i in $(seq 1 "$VMS"); do start_vm "$i"; done
  for i in $(seq 1 "$VMS"); do wait_ssh "$i"; done
  write_inventory
  log "VMs are up. Next: ./vms.sh provision"
}

cmd_provision() {
  require_tools
  [ -f "$ROOT/inventory.ini" ] || { echo "no inventory — run ./vms.sh up first" >&2; exit 1; }
  mkdir -p "$ROOT/build"
  log "building linux agent binary for the VMs"
  (cd "$REPO_ROOT" && CGO_ENABLED=0 go build -o "$ROOT/build/pz-agent" ./apps/pz-agent/cmd/agent)
  ensure_backend
  log "running Ansible (update, steamcmd, PZ server, mods, agent) — PZ download is ~4 GB per VM on first run"
  ansible-playbook -i "$ROOT/inventory.ini" "$REPO_ROOT/deploy/vms/ansible/playbook.yml"
}

cmd_status() {
  local i
  echo
  echo "─── VMs ───"
  for i in $(seq 1 "$VMS"); do
    local state=stopped ssh_state=-
    vm_running "$i" && state="running (pid $(cat "$(vm_pidfile "$i")"))"
    if [ "$state" != stopped ]; then
      ssh "${SSH_OPTS[@]}" -p "$(vm_ssh_port "$i")" pz@127.0.0.1 true 2>/dev/null && ssh_state=reachable || ssh_state=no-ssh
    fi
    echo "  $(vm_name "$i")  $state  ssh:127.0.0.1:$(vm_ssh_port "$i") ($ssh_state)  game:udp/$(vm_game_port "$i")"
  done
  echo "─── backend ───"
  if curl -fsS --max-time 3 "$BACKEND_URL/api/v1/health" >/dev/null 2>&1; then
    echo "  $BACKEND_URL  $(curl -fsS "$BACKEND_URL/api/v1/health" | tr -d ' \n')"
    curl -fsS "$BACKEND_URL/api/v1/agents" 2>/dev/null | python3 -c '
import json,sys
for a in json.load(sys.stdin)["agents"]:
    print("  agent {serverId}  {status}  mods={modCount}  lastSeen={lastSeen}".format(**a))' 2>/dev/null || true
  else
    echo "  backend not running ($BACKEND_URL) — ./vms.sh up starts it"
  fi
  echo
}

cmd_ssh() {
  local i="${SSH_ARG:-1}"
  exec ssh "${SSH_OPTS[@]}" -p "$(vm_ssh_port "$i")" pz@127.0.0.1
}

cmd_down() {
  local i
  for i in $(seq 1 "$VMS"); do
    vm_running "$i" || continue
    log "powering off $(vm_name "$i")"
    ssh "${SSH_OPTS[@]}" -p "$(vm_ssh_port "$i")" pz@127.0.0.1 'sudo poweroff' 2>/dev/null || true
  done
  # wait for graceful shutdown, then force-kill leftovers
  local tries=24
  while [ $tries -gt 0 ]; do
    local alive=0
    for i in $(seq 1 "$VMS"); do vm_running "$i" && alive=1; done
    [ $alive = 0 ] && break
    sleep 5; tries=$((tries - 1))
  done
  for i in $(seq 1 "$VMS"); do
    if vm_running "$i"; then
      log "force-killing $(vm_name "$i")"
      kill "$(cat "$(vm_pidfile "$i")")" 2>/dev/null || true
    fi
    rm -f "$(vm_pidfile "$i")"
  done
  if [ -f "$ROOT/run/backend.pid" ]; then
    kill "$(cat "$ROOT/run/backend.pid")" 2>/dev/null || true
    rm -f "$ROOT/run/backend.pid"
    log "backend stopped"
  fi
  log "all down (disks kept — next up reuses them)"
}

cmd_destroy() {
  cmd_down
  local i
  for i in $(seq 1 "$VMS"); do rm -f "$(vm_disk "$i")" "$(vm_seed "$i")"; done
  log "disks deleted — next up creates fresh VMs"
}

case "$COMMAND" in
  up)        cmd_up ;;
  provision) cmd_provision ;;
  status)    cmd_status ;;
  ssh)       cmd_ssh ;;
  down)      cmd_down ;;
  destroy)   cmd_destroy ;;
  *) echo "usage: vms.sh up|provision|status|ssh N|down|destroy [options]" >&2; exit 2 ;;
esac
