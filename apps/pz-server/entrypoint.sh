#!/bin/bash
set -e

SERVER_NAME="${PZ_SERVER_NAME:-pz-test}"
MODS_DIR="/mods"
MOD_COUNT="${PZ_MOD_COUNT:-5}"

echo "=== PZ Agent Test Node ==="
echo "Server Name:  $SERVER_NAME"
echo "Backend:      $PZ_BACKEND"
echo "Interval:     $PZ_AGENT_INTERVAL"
echo ""

# ── Create fake mods if the volume is empty ──
if [ -z "$(ls -A $MODS_DIR 2>/dev/null)" ]; then
  echo "Generating $MOD_COUNT fake mods in $MODS_DIR ..."
  for i in $(seq 1 $MOD_COUNT); do
    MOD_DIR="$MODS_DIR/test-mod-${SERVER_NAME}-$i"
    mkdir -p "$MOD_DIR"
    printf "name=Test Mod %d\nversion=1.%d\nworkshopId=10000%d\n" "$i" "$i" "$i" > "$MOD_DIR/mod.info"
    dd if=/dev/urandom of="$MOD_DIR/data.bin" bs=1024 count=$((i * 8)) 2>/dev/null
    echo "  + test-mod-${SERVER_NAME}-$i"
  done
  echo ""
fi

# ── Start PZ Agent ──
echo "Starting agent → $PZ_BACKEND"
exec /usr/local/bin/pz-agent \
  -server "$SERVER_NAME" \
  -backend "$PZ_BACKEND" \
  -mods "$MODS_DIR" \
  -interval "$PZ_AGENT_INTERVAL"
