# PZ Dedicated Server + Agent — for test environments
# Uses SteamCMD to install Project Zomboid Dedicated Server (appid 380870)
# Then runs PZ server + agent side-by-side via entrypoint script.
#
# Build:  docker build -t pz-server -f apps/pz-server/Dockerfile .
# Run:    docker run -d -p 16261:16261/udp -p 16262:16262 \
#           -e PZ_BACKEND=http://backend:8080 \
#           -e PZ_SERVER_NAME=test-server-1 \
#           pz-server

FROM golang:1.23-alpine AS agent-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/agent ./apps/pz-agent/cmd/agent

# ── Runtime ──
FROM steamcmd/steamcmd:ubuntu-24

USER root

# Required packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl procps jq dos2unix \
    && rm -rf /var/lib/apt/lists/*

# Create pz user
RUN useradd -ms /bin/bash pzuser

USER pzuser
WORKDIR /home/pzuser

# Install PZ Dedicated Server via SteamCMD
RUN steamcmd +force_install_dir /home/pzuser/pz-server \
    +login anonymous \
    +app_update 380870 validate \
    +quit

# Copy agent binary
COPY --from=agent-builder /out/agent /usr/local/bin/pz-agent

# Copy entrypoint
COPY apps/pz-server/entrypoint.sh /home/pzuser/entrypoint.sh

USER root
RUN chmod +x /home/pzuser/entrypoint.sh && dos2unix /home/pzuser/entrypoint.sh

USER pzuser

# PZ Server ports: 16261 UDP (game), 16262 TCP (RCON/direct)
EXPOSE 16261/udp 16262

# Environment defaults
ENV PZ_BACKEND=http://backend:8080 \
    PZ_SERVER_NAME=pz-test \
    PZ_SERVER_MEMORY=2048m \
    PZ_ADMIN_PASSWORD=admin \
    PZ_AGENT_INTERVAL=60s

VOLUME ["/home/pzuser/Zomboid"]

ENTRYPOINT ["/home/pzuser/entrypoint.sh"]
