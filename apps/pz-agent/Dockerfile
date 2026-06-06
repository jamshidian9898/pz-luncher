# Stage 1 — build
FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" \
    -o /out/agent ./apps/pz-agent/cmd/agent

# Stage 2 — minimal runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    addgroup -S pz && adduser -S pz -G pz

WORKDIR /app
COPY --from=builder /out/agent /app/agent

RUN mkdir -p /mods && chown pz:pz /mods

USER pz

# Required env:
#   PZ_AGENT_SERVER  — serverId registered in the Backend
#   PZ_AGENT_BACKEND — e.g. http://backend:8080
#   PZ_AGENT_TOKEN   — pre-issued token (leave empty for auto-register)
#   PZ_AGENT_MODS    — path to mod directory (default /mods)

ENTRYPOINT ["/app/agent"]
CMD ["-server", "${PZ_AGENT_SERVER}", \
     "-backend", "${PZ_AGENT_BACKEND}", \
     "-mods", "/mods"]
