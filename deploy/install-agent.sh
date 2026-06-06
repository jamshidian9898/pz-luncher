#!/bin/sh
# PZ Agent Installer
# Usage:
#   curl -fsSL http://YOUR_BACKEND/install-agent.sh | \
#     PZ_BACKEND=http://YOUR_BACKEND \
#     PZ_SERVER=my-server \
#     PZ_TOKEN=your-token \
#     sh
#
# Or interactively:
#   sh install-agent.sh
set -e

PZ_BACKEND="${PZ_BACKEND:-}"
PZ_SERVER="${PZ_SERVER:-}"
PZ_TOKEN="${PZ_TOKEN:-}"
PZ_MODS="${PZ_MODS:-/srv/pz-mods}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
SERVICE_NAME="pz-agent"

ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  *)
    echo "ERROR: Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

echo "=== PZ Agent Installer ==="
echo "Architecture: linux-$ARCH"

# Prompt for missing values
if [ -z "$PZ_BACKEND" ]; then
  printf "Backend URL (e.g. http://192.168.1.10:8080): "
  read -r PZ_BACKEND
fi
if [ -z "$PZ_SERVER" ]; then
  printf "Server ID: "
  read -r PZ_SERVER
fi
if [ -z "$PZ_TOKEN" ]; then
  printf "Agent token (leave empty for auto-register): "
  read -r PZ_TOKEN
fi

# Download binary
BINARY_URL="${PZ_BACKEND}/releases/agent-linux-${ARCH}"
echo ""
echo "Downloading agent from ${BINARY_URL} ..."
curl -fsSL -o "/tmp/pz-agent" "$BINARY_URL"
chmod +x /tmp/pz-agent
mv /tmp/pz-agent "${INSTALL_DIR}/pz-agent"
echo "Installed: ${INSTALL_DIR}/pz-agent"

# Create mods directory
mkdir -p "$PZ_MODS"

# Write systemd service
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
if command -v systemctl >/dev/null 2>&1 && [ "$(id -u)" = "0" ]; then
  cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=PZ Agent Content Publisher
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/pz-agent -server ${PZ_SERVER} -backend ${PZ_BACKEND} -mods ${PZ_MODS} -interval 60s
Restart=on-failure
RestartSec=10s
Environment=PZ_AGENT_TOKEN=${PZ_TOKEN}

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME"
  systemctl start "$SERVICE_NAME"

  echo ""
  echo "=== Agent service installed and started ==="
  echo "  Status:  systemctl status ${SERVICE_NAME}"
  echo "  Logs:    journalctl -u ${SERVICE_NAME} -f"
else
  echo ""
  echo "=== Manual start (systemd not available or not root) ==="
  ENV_ARGS=""
  [ -n "$PZ_TOKEN" ] && ENV_ARGS="PZ_AGENT_TOKEN=${PZ_TOKEN}"
  echo "  ${ENV_ARGS} ${INSTALL_DIR}/pz-agent -server ${PZ_SERVER} -backend ${PZ_BACKEND} -mods ${PZ_MODS} -interval 60s"
fi

echo ""
echo "Done."
