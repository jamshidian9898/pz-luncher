#!/bin/sh
# PZ Agent Installer
#
# Minimal usage (auto-detects PZ server, mods, and auto-registers):
#   curl -fsSL http://YOUR_BACKEND/install-agent.sh | PZ_BACKEND=http://YOUR_BACKEND sh
#
# Full usage:
#   PZ_BACKEND=http://192.168.1.10:8080 PZ_SERVER=myserver PZ_TOKEN=xxx sh install-agent.sh
#
# The agent will auto-detect the PZ server process, server name, and mods
# directory if -server and -mods are not provided.
set -e

PZ_BACKEND="${PZ_BACKEND:-}"
PZ_SERVER="${PZ_SERVER:-}"
PZ_TOKEN="${PZ_TOKEN:-}"
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

# Only backend URL is required — server and mods are auto-detected.
if [ -z "$PZ_BACKEND" ]; then
  printf "Backend URL (e.g. http://192.168.1.10:8080): "
  read -r PZ_BACKEND </dev/tty
fi

# Download binary
BINARY_URL="${PZ_BACKEND}/releases/agent-linux-${ARCH}"
echo ""
echo "Downloading agent from ${BINARY_URL} ..."
curl -fsSL -o "/tmp/pz-agent" "$BINARY_URL"
chmod +x /tmp/pz-agent
mv /tmp/pz-agent "${INSTALL_DIR}/pz-agent"
echo "Installed: ${INSTALL_DIR}/pz-agent"

# Build ExecStart command — only add flags for explicitly provided values.
# Agent auto-detects -server and -mods if omitted.
AGENT_ARGS="-backend ${PZ_BACKEND} -interval 60s"
[ -n "$PZ_SERVER" ] && AGENT_ARGS="${AGENT_ARGS} -server ${PZ_SERVER}"

ENV_LINE=""
[ -n "$PZ_TOKEN" ] && ENV_LINE="Environment=PZ_AGENT_TOKEN=${PZ_TOKEN}"

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
ExecStart=${INSTALL_DIR}/pz-agent ${AGENT_ARGS}
Restart=on-failure
RestartSec=10s
${ENV_LINE}

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME"
  systemctl start "$SERVICE_NAME"

  echo ""
  echo "=== Agent service installed and started ==="
  echo "  The agent will auto-detect your PZ server and mods directory."
  echo "  Status:  systemctl status ${SERVICE_NAME}"
  echo "  Logs:    journalctl -u ${SERVICE_NAME} -f"
else
  echo ""
  echo "=== Manual start (systemd not available or not root) ==="
  RUN_CMD="${INSTALL_DIR}/pz-agent ${AGENT_ARGS}"
  [ -n "$PZ_TOKEN" ] && RUN_CMD="PZ_AGENT_TOKEN=${PZ_TOKEN} ${RUN_CMD}"
  echo "  ${RUN_CMD}"
fi

echo ""
echo "Done."
