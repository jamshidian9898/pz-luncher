# Stage 1 — build
FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" \
    -o /out/backend ./apps/backend/cmd/backend

# Stage 2 — minimal runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S pz && adduser -S pz -G pz

WORKDIR /app
COPY --from=builder /out/backend /app/backend
COPY --chown=pz:pz apps/backend/registry.json /app/registry.json
COPY --chown=pz:pz fixtures/ /app/fixtures/

RUN mkdir -p /data && chown pz:pz /data

USER pz

EXPOSE 8080

ENTRYPOINT ["/app/backend"]
CMD ["-addr", ":8080", \
     "-registry", "/app/registry.json", \
     "-store", "/data/store", \
     "-fixtures", "/app/fixtures"]
