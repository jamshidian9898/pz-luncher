#!/usr/bin/env bash
#
# PZ Platform — single-Linux test lab (Ubuntu / WSL).
#
# Everything runs on one machine: the backend, N Project Zomboid dedicated
# servers (steamcmd), one agent per server, and a verify step that proves the
# full content loop:
#
#   agent scans mods → deterministic-zip blobs + versioned manifest → backend
#   → join-cli POST /join → downloaded, extracted, byte-identical profile
#   → game launched with -cachedir=<profile>
#
# Service management is automatic:
#   - systemd present (Ubuntu server, WSL2 with systemd=true) → system units
#   - no systemd (default WSL)                                → pid-file mode
#
# Usage:
#   ./lab.sh install [--skip-game-servers]   # build, seed, install services
#   ./lab.sh up                              # start backend, agents, servers
#   ./lab.sh verify                          # prove the loop; exit 0 = PASS
#   ./lab.sh status                          # one-screen health view
#   ./lab.sh down                            # stop everything
#   ./lab.sh clean                           # down + remove services
#
# Options: --root DIR (default ~/pz-lab)  --servers N (default 2)
#          --backend-port P (default 8080)  --skip-game-servers  --no-systemd
set -euo pipefail

COMMAND=${1:-status}
[ $# -gt 0 ] && shift

ROOT="${PZ_LAB_ROOT:-$HOME/pz-lab}"
SERVERS=2
BACKEND_PORT=8080
SKIP_GAME_SERVERS=0
NO_SYSTEMD=0

while [ $# -gt 0 ]; do
  case "$1" in
    --root) ROOT="$2"; shift 2 ;;
    --servers) SERVERS="$2"; shift 2 ;;
    --backend-port) BACKEND_PORT="$2"; shift 2 ;;
    --skip-game-servers) SKIP_GAME_SERVERS=1; shift ;;
    --no-systemd) NO_SYSTEMD=1; shift ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKEND_URL="http://localhost:$BACKEND_PORT"
PZ_APP_ID=380870
STEAMCMD_TGZ="https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz"

log()  { printf '\033[36m[lab]\033[0m %s\n' "$*"; }
ok()   { printf '  \033[32mPASS\033[0m  %s\n' "$*"; }
fail() { printf '  \033[31mFAIL\033[0m  %s\n' "$*"; }

server_name()  { echo "pzlab$1"; }
server_data()  { echo "$ROOT/data/srv$1"; }
server_mods()  { echo "$ROOT/data/srv$1/mods"; }
server_port()  { echo $((16261 + ($1 - 1) * 2)); }

use_systemd() {
  [ "$NO_SYSTEMD" = 1 ] && return 1
  [ -d /run/systemd/system ] && command -v systemctl >/dev/null
}

SUDO=""
[ "$(id -u)" != 0 ] && SUDO="sudo"

# ───────────────────────── install ─────────────────────────

build_binaries() {
  log "building lab binaries from $REPO_ROOT"
  command -v go >/dev/null || { echo "Go not found on PATH. Install Go 1.23+ and retry." >&2; exit 1; }
  mkdir -p "$ROOT/bin" "$ROOT/fakegame"
  (cd "$REPO_ROOT" &&
    go build -o "$ROOT/bin/pz-backend" ./apps/backend/cmd/backend &&
    go build -o "$ROOT/bin/pz-agent" ./apps/pz-agent/cmd/agent &&
    go build -o "$ROOT/bin/join-cli" ./apps/join-cli &&
    go build -o "$ROOT/fakegame/ProjectZomboid64" ./tools/fakegame)
}

install_steamcmd() {
  local dir="$ROOT/steamcmd"
  if [ -x "$dir/steamcmd.sh" ]; then log "steamcmd already installed"; return; fi
  log "installing steamcmd"
  if ! dpkg -s lib32gcc-s1 >/dev/null 2>&1; then
    log "installing lib32gcc-s1 (32-bit runtime for steamcmd)"
    $SUDO apt-get update -qq && $SUDO apt-get install -y -qq lib32gcc-s1
  fi
  mkdir -p "$dir"
  curl -fsSL "$STEAMCMD_TGZ" | tar -xz -C "$dir"
}

install_pz_server() {
  install_steamcmd
  local dest="$ROOT/pzserver"
  log "installing PZ Dedicated Server (app $PZ_APP_ID) into $dest — several GB, be patient"
  # steamcmd sometimes exits non-zero on first self-update; retry once.
  "$ROOT/steamcmd/steamcmd.sh" +force_install_dir "$dest" +login anonymous +app_update $PZ_APP_ID validate +quit ||
    "$ROOT/steamcmd/steamcmd.sh" +force_install_dir "$dest" +login anonymous +app_update $PZ_APP_ID validate +quit
  [ -f "$dest/start-server.sh" ] || { echo "PZ server install failed: start-server.sh not found in $dest" >&2; exit 1; }
}

seed_sample_mods() {
  local i=$1 mods; mods="$(server_mods "$i")"
  if [ -d "$mods" ] && [ -n "$(ls -A "$mods" 2>/dev/null)" ]; then return; fi
  log "seeding sample mods for $(server_name "$i")"
  mkdir -p "$mods/LabMod$i/media/lua"
  printf 'name=LabMod%s\nid=LabMod%s\n' "$i" "$i" > "$mods/LabMod$i/mod.info"
  printf "print('lab mod %s for %s')\n" "$i" "$(server_name "$i")" > "$mods/LabMod$i/media/lua/init.lua"
}

backend_args() {
  echo "-addr :$BACKEND_PORT -registry $ROOT/data/backend/registry.json -store $ROOT/data/backend/store -content-registry $ROOT/data/backend/content-registry.json -fixtures $ROOT/data/backend/fixtures"
}

agent_args() {
  local i=$1
  echo "-server $(server_name "$i") -mods $(server_mods "$i") -backend $BACKEND_URL -interval 60s"
}

install_systemd_units() {
  log "installing systemd units (running as user $USER)"
  $SUDO tee /etc/systemd/system/pz-backend.service >/dev/null <<EOF
[Unit]
Description=PZ Platform Backend (lab)
After=network.target

[Service]
User=$USER
ExecStart=$ROOT/bin/pz-backend $(backend_args)
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
  local i
  for i in $(seq 1 "$SERVERS"); do
    $SUDO tee "/etc/systemd/system/pz-agent$i.service" >/dev/null <<EOF
[Unit]
Description=PZ Platform Agent for $(server_name "$i") (lab)
After=pz-backend.service

[Service]
User=$USER
ExecStart=$ROOT/bin/pz-agent $(agent_args "$i")
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
  done
  $SUDO systemctl daemon-reload
  $SUDO systemctl enable pz-backend $(seq -f 'pz-agent%.0f' 1 "$SERVERS") >/dev/null 2>&1
}

cmd_install() {
  mkdir -p "$ROOT/logs" "$ROOT/run" "$ROOT/data/backend/store" "$ROOT/launcher"
  [ -f "$ROOT/data/backend/registry.json" ] || echo '{"servers":[]}' > "$ROOT/data/backend/registry.json"
  build_binaries
  local i
  for i in $(seq 1 "$SERVERS"); do seed_sample_mods "$i"; done
  [ "$SKIP_GAME_SERVERS" = 1 ] || install_pz_server
  if use_systemd; then install_systemd_units; else log "no systemd — will use pid-file mode"; fi
  log "install complete. Next: ./lab.sh up"
}

# ───────────────────────── up / down ─────────────────────────

start_bg() { # name, logfile, cmd...
  local name=$1 logfile=$2; shift 2
  local pidfile="$ROOT/run/$name.pid"
  if [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
    log "$name already running (pid $(cat "$pidfile"))"; return
  fi
  log "starting $name"
  setsid nohup "$@" >> "$logfile" 2>&1 &
  echo $! > "$pidfile"
}

stop_bg() {
  local name=$1 pidfile="$ROOT/run/$1.pid"
  [ -f "$pidfile" ] || return 0
  local pid; pid="$(cat "$pidfile")"
  if kill -0 "$pid" 2>/dev/null; then
    log "stopping $name (pid $pid)"
    kill -TERM -- "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
    sleep 1
    kill -KILL -- "-$pid" 2>/dev/null || true
  fi
  rm -f "$pidfile"
}

start_game_server() {
  local i=$1
  [ -f "$ROOT/pzserver/start-server.sh" ] || { log "PZ server not installed — skipping $(server_name "$i")"; return; }
  start_bg "srv$i" "$ROOT/logs/srv$i.log" \
    "$ROOT/pzserver/start-server.sh" \
    -cachedir="$(server_data "$i")" \
    -servername "$(server_name "$i")" \
    -port "$(server_port "$i")" \
    -adminusername admin -adminpassword 'pzlab-admin-1!'
}

cmd_up() {
  local i
  if use_systemd; then
    log "starting services via systemd"
    $SUDO systemctl start pz-backend
    for i in $(seq 1 "$SERVERS"); do $SUDO systemctl start "pz-agent$i"; done
  else
    start_bg backend "$ROOT/logs/pz-backend.log" "$ROOT/bin/pz-backend" $(backend_args)
    sleep 1
    for i in $(seq 1 "$SERVERS"); do
      start_bg "agent$i" "$ROOT/logs/pz-agent$i.log" "$ROOT/bin/pz-agent" $(agent_args "$i")
    done
  fi
  if [ "$SKIP_GAME_SERVERS" != 1 ]; then
    for i in $(seq 1 "$SERVERS"); do start_game_server "$i"; done
  fi
  log "lab is up. Verify with: ./lab.sh verify"
}

cmd_down() {
  local i
  for i in $(seq 1 "$SERVERS"); do stop_bg "srv$i"; done
  if use_systemd; then
    for i in $(seq 1 "$SERVERS"); do $SUDO systemctl stop "pz-agent$i" 2>/dev/null || true; done
    $SUDO systemctl stop pz-backend 2>/dev/null || true
  else
    for i in $(seq 1 "$SERVERS"); do stop_bg "agent$i"; done
    stop_bg backend
  fi
  log "lab is down"
}

cmd_clean() {
  cmd_down
  if use_systemd; then
    local i
    for i in $(seq 1 "$SERVERS"); do
      $SUDO systemctl disable "pz-agent$i" >/dev/null 2>&1 || true
      $SUDO rm -f "/etc/systemd/system/pz-agent$i.service"
    done
    $SUDO systemctl disable pz-backend >/dev/null 2>&1 || true
    $SUDO rm -f /etc/systemd/system/pz-backend.service
    $SUDO systemctl daemon-reload
  fi
  log "services removed. Lab data kept at $ROOT (delete manually if desired)"
}

# ───────────────────────── status / verify ─────────────────────────

json_get() { # url, python expression over parsed `d`
  curl -fsS --max-time 5 "$1" | python3 -c "import json,sys; d=json.load(sys.stdin); print($2)"
}

cmd_status() {
  local i
  echo
  echo "─── services ───"
  if use_systemd; then
    systemctl --no-pager list-units 'pz-*' 2>/dev/null | sed -n '2,10p' || true
  else
    for name in backend $(seq -f 'agent%.0f' 1 "$SERVERS"); do
      local pidfile="$ROOT/run/$name.pid" state=stopped
      [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null && state="running (pid $(cat "$pidfile"))"
      echo "  $name  $state"
    done
  fi
  echo "─── game servers ───"
  for i in $(seq 1 "$SERVERS"); do
    local pidfile="$ROOT/run/srv$i.pid" state=stopped
    [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null && state="running (pid $(cat "$pidfile"))"
    echo "  $(server_name "$i")  port $(server_port "$i")  $state"
  done
  echo "─── backend ───"
  if curl -fsS --max-time 3 "$BACKEND_URL/api/v1/health" >/dev/null 2>&1; then
    echo "  $BACKEND_URL  $(curl -fsS "$BACKEND_URL/api/v1/health" | tr -d ' \n')"
    curl -fsS "$BACKEND_URL/api/v1/agents" | python3 -c '
import json,sys
for a in json.load(sys.stdin)["agents"]:
    print("  agent {serverId}  {status}  mods={modCount}  lastSeen={lastSeen}".format(**a))' 2>/dev/null || true
  else
    echo "  backend unreachable at $BACKEND_URL"
  fi
  echo
}

cmd_verify() {
  local failures=0 i
  log "verifying the end-to-end content loop against $BACKEND_URL"

  # 1. backend health
  if [ "$(json_get "$BACKEND_URL/api/v1/health" "d['status']" 2>/dev/null)" = ok ]; then
    ok "backend /health"
  else fail "backend unreachable or unhealthy"; failures=$((failures+1)); fi

  # 2. all servers registered (agents auto-register them)
  for i in $(seq 1 "$SERVERS"); do
    local name; name="$(server_name "$i")"
    if json_get "$BACKEND_URL/api/v1/servers" "any(s['id']=='$name' for s in d['servers'])" 2>/dev/null | grep -q True; then
      ok "server $name registered"
    else fail "server $name not in registry (agent not synced yet?)"; failures=$((failures+1)); fi
  done

  # 3. agents heartbeating
  for i in $(seq 1 "$SERVERS"); do
    local name status; name="$(server_name "$i")"
    status="$(json_get "$BACKEND_URL/api/v1/agents" "next((a['status'] for a in d['agents'] if a['serverId']=='$name'),'missing')" 2>/dev/null || echo unreachable)"
    if [ "$status" = online ]; then ok "agent for $name online"
    else fail "agent for $name is $status"; failures=$((failures+1)); fi
  done

  # 4. manifests published
  for i in $(seq 1 "$SERVERS"); do
    local name latest; name="$(server_name "$i")"
    latest="$(json_get "$BACKEND_URL/api/v1/manifests/$name/history" "max(v['version'] for v in d['versions'])" 2>/dev/null || echo 0)"
    if [ "$latest" -ge 1 ] 2>/dev/null; then ok "manifest for $name published (v$latest)"
    else fail "no manifest versions for $name"; failures=$((failures+1)); fi
  done

  # 5. join through the same pipeline the launcher UI uses
  local primary; primary="$(server_name 1)"
  if PZ_LAUNCHER_ROOT="$ROOT/launcher" "$ROOT/bin/join-cli" -server "$primary" -backend "$BACKEND_URL" \
      > "$ROOT/logs/verify-join.log" 2>&1; then
    ok "join-cli -backend completed for $primary"
  else fail "join-cli failed (see logs/verify-join.log)"; failures=$((failures+1)); fi

  # 6. launch flow passes profile isolation to the (fake) game
  if [ -x "$ROOT/fakegame/ProjectZomboid64" ]; then
    rm -f "$ROOT/fakegame/launch-args.txt"
    PZ_LAUNCHER_ROOT="$ROOT/launcher" PZ_GAME_PATH="$ROOT/fakegame" \
      "$ROOT/bin/join-cli" -server "$primary" -backend "$BACKEND_URL" -launch \
      > "$ROOT/logs/verify-launch.log" 2>&1 || true
    sleep 3
    if grep -q '^-cachedir=' "$ROOT/fakegame/launch-args.txt" 2>/dev/null; then
      ok "game launched with -cachedir pointing at the profile"
    else fail "fake game did not receive -cachedir (see logs/verify-launch.log)"; failures=$((failures+1)); fi
  fi

  # 7. extracted profile is byte-identical to the server's mods
  local src dst; src="$(server_mods 1)"; dst="$ROOT/launcher/profiles/$primary/mods"
  if [ -d "$dst" ] && diff -r --exclude=.pz-extracted "$src" "$dst" >/dev/null 2>&1; then
    ok "profile content byte-identical to $primary server mods"
  else fail "extracted profile differs from server mods ($dst)"; failures=$((failures+1)); fi

  echo
  if [ "$failures" -eq 0 ]; then log "VERIFY PASSED — full loop proven on this machine"; exit 0
  else log "VERIFY FAILED — $failures check(s) failed"; exit 1; fi
}

case "$COMMAND" in
  install) cmd_install ;;
  up)      cmd_up ;;
  down)    cmd_down ;;
  status)  cmd_status ;;
  verify)  cmd_verify ;;
  clean)   cmd_clean ;;
  *) echo "usage: lab.sh install|up|down|status|verify|clean [options]" >&2; exit 2 ;;
esac
