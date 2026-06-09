#!/bin/bash
set -e

PZ_DIR="/home/pzuser/pz-server"
ZOMBOID_DIR="/home/pzuser/Zomboid"
SERVER_NAME="${PZ_SERVER_NAME:-pz-test}"

echo "=== PZ Test Server ==="
echo "Server Name:  $SERVER_NAME"
echo "Backend:      $PZ_BACKEND"
echo "Memory:       $PZ_SERVER_MEMORY"
echo ""

# ── Create server config if it doesn't exist ──
mkdir -p "$ZOMBOID_DIR/Server"
INI_FILE="$ZOMBOID_DIR/Server/${SERVER_NAME}.ini"

if [ ! -f "$INI_FILE" ]; then
  echo "Creating default server config: $INI_FILE"
  cat > "$INI_FILE" <<EOF
DefaultPort=16261
MaxPlayers=16
PVP=true
PauseEmpty=true
Open=true
ServerName=${SERVER_NAME}
Password=
AdminPassword=${PZ_ADMIN_PASSWORD}
PublicName=${SERVER_NAME}
PublicDescription=PZ Test Server (auto-provisioned)
ResetID=0
Map=Muldraugh, KY
EOF
fi

# ── Create sample mods dir with some test content ──
MODS_DIR="$ZOMBOID_DIR/mods"
mkdir -p "$MODS_DIR"

# Create a few fake mods for testing if none exist
if [ -z "$(ls -A $MODS_DIR 2>/dev/null)" ]; then
  echo "Creating sample test mods..."
  for i in 1 2 3; do
    MOD_DIR="$MODS_DIR/test-mod-$i"
    mkdir -p "$MOD_DIR"
    echo "Test mod $i - version 1.0" > "$MOD_DIR/mod.info"
    # Create some random content so SHA256 differs per mod
    dd if=/dev/urandom of="$MOD_DIR/data.bin" bs=1024 count=$((i * 10)) 2>/dev/null
    echo "  Created test-mod-$i ($(du -sh $MOD_DIR | cut -f1))"
  done
fi

# ── Start PZ Agent in background ──
echo ""
echo "Starting PZ Agent..."
AGENT_ARGS="-backend $PZ_BACKEND -interval $PZ_AGENT_INTERVAL"
[ -n "$PZ_SERVER_NAME" ] && AGENT_ARGS="$AGENT_ARGS -server $PZ_SERVER_NAME"
[ -n "$PZ_AGENT_TOKEN" ] && export PZ_AGENT_TOKEN

/usr/local/bin/pz-agent $AGENT_ARGS &
AGENT_PID=$!
echo "Agent started (PID $AGENT_PID)"

# ── Start PZ Dedicated Server ──
echo ""
echo "Starting PZ Dedicated Server..."
cd "$PZ_DIR"

# PZ server start script
if [ -f "./start-server.sh" ]; then
  chmod +x ./start-server.sh
  exec bash -c "
    # Trap to kill agent when server exits
    trap 'kill $AGENT_PID 2>/dev/null; exit' EXIT INT TERM
    ./start-server.sh \
      -servername \"$SERVER_NAME\" \
      -Xmx$PZ_SERVER_MEMORY \
      -Xms512m &
    PZ_PID=\$!
    wait \$PZ_PID
  "
else
  echo "WARNING: start-server.sh not found. Running agent-only mode for testing."
  echo "Agent will keep running and publishing mods to backend."
  echo ""
  # Keep container alive with agent only
  wait $AGENT_PID
fi
