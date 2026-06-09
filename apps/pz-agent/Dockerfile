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

# Environment:
#   PZ_AGENT_TOKEN — pre-issued token (leave empty for auto-register)
# All flags are optional — agent auto-detects server name and mods dir.

ENTRYPOINT ["/app/agent"]
CMD ["-backend", "http://localhost:8080", "-mods", "/mods"]
