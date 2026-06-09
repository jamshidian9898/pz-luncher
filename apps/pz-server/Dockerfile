# PZ Agent Test Node — simulates a PZ server for local testing
# Does NOT install the real PZ server (requires Steam login).
# Instead, creates fake mod content and runs the agent to publish to backend.
#
# Build:  docker build -t pz-server -f apps/pz-server/Dockerfile .
# Run:    docker run -d \
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
FROM alpine:3.20

RUN apk add --no-cache ca-certificates bash && \
    addgroup -S pz && adduser -S pz -G pz -s /bin/bash

# Copy agent binary
COPY --from=agent-builder /out/agent /usr/local/bin/pz-agent

# Copy entrypoint
COPY apps/pz-server/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN mkdir -p /mods && chown pz:pz /mods

USER pz

# Environment defaults
ENV PZ_BACKEND=http://backend:8080 \
    PZ_SERVER_NAME=pz-test \
    PZ_AGENT_INTERVAL=30s \
    PZ_MOD_COUNT=5

VOLUME ["/mods"]

ENTRYPOINT ["/entrypoint.sh"]
