# Content Registry Deployment Guide

Deploy a self-hosted Content Registry for v2.1.0 (RFC-0059).

---

## Requirements

- **OS**: Linux (Ubuntu 20.04 LTS or later recommended)
- **Go**: 1.23+
- **Disk**: Minimum 50 GB (for initial binary cache)
- **RAM**: 2 GB
- **Network**: Static IP, ports 80 & 443 open, reachable from target region (Iran, etc.)
- **TLS**: Let's Encrypt certificate or self-signed

---

## Option 1: Docker (Recommended)

### 1. Install Docker & Docker Compose

```bash
sudo apt update
sudo apt install -y docker.io docker-compose
sudo usermod -aG docker $USER
```

### 2. Create Deployment Directory

```bash
mkdir -p ~/pz-registry/{data,config}
cd ~/pz-registry
```

### 3. Create docker-compose.yml

```yaml
version: '3.8'
services:
  registry:
    image: golang:1.23-alpine as builder
    container_name: pz-registry
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
      - "443:443"
    volumes:
      - ./data:/app/data          # Content storage
      - ./config:/app/config      # Registry metadata
      - ./certs:/app/certs        # TLS certificates
    environment:
      - REGISTRY_ADDR=:8080
      - REGISTRY_DATA_PATH=/app/data
      - REGISTRY_CONFIG_PATH=/app/config/registry.json
      - REGISTRY_TLS_CERT=/app/certs/cert.pem
      - REGISTRY_TLS_KEY=/app/certs/key.pem
    restart: always
```

### 4. Build Docker Image

Clone the repo and build:

```bash
cd ~/pz-registry
git clone https://github.com/yourusername/pz-luncher.git
cd pz-luncher

docker build -f apps/backend/Dockerfile.registry -t pz-registry:latest .
```

Create `Dockerfile.registry`:

```dockerfile
FROM golang:1.23-alpine as builder
WORKDIR /src
COPY go.mod go.sum ./
COPY apps/backend ./apps/backend
COPY libs ./libs
RUN go build -o registry ./apps/backend/cmd/registry

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /src/registry /usr/local/bin/
ENTRYPOINT ["registry"]
```

### 5. Start Registry

```bash
docker-compose up -d
docker-compose logs -f  # Watch logs
```

---

## Option 2: Manual Installation (Linux)

### 1. Install Go 1.23

```bash
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### 2. Clone & Build

```bash
mkdir -p ~/pz-registry
cd ~/pz-registry
git clone https://github.com/yourusername/pz-luncher.git
cd pz-luncher

go build -o ~/pz-registry/registry ./apps/backend/cmd/registry
```

### 3. Create Systemd Service

```bash
sudo tee /etc/systemd/system/pz-registry.service > /dev/null <<EOF
[Unit]
Description=PZ Content Registry
After=network.target

[Service]
Type=simple
User=pz-registry
WorkingDirectory=/opt/pz-registry
ExecStart=/opt/pz-registry/registry \
  -addr :8080 \
  -store /opt/pz-registry/data \
  -registry /opt/pz-registry/config/registry.json
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable pz-registry
sudo systemctl start pz-registry
```

### 4. Verify

```bash
curl http://localhost:8080/api/v1/registry/catalog?game=pz
```

---

## TLS/SSL Setup

### Self-Signed Certificate (for testing)

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

### Let's Encrypt (Production)

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot certonly --standalone -d registry.yourdomain.com

# Copy certs
sudo cp /etc/letsencrypt/live/registry.yourdomain.com/fullchain.pem certs/cert.pem
sudo cp /etc/letsencrypt/live/registry.yourdomain.com/privkey.pem certs/key.pem
```

### Nginx Reverse Proxy

```nginx
server {
    listen 443 ssl http2;
    server_name registry.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/registry.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/registry.yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Large file upload support
        client_max_body_size 10G;
        proxy_request_buffering off;
    }
}

server {
    listen 80;
    server_name registry.yourdomain.com;
    return 301 https://$server_name$request_uri;
}
```

Reload Nginx:

```bash
sudo systemctl reload nginx
```

---

## Configuration

### Registry Metadata File

`config/registry.json` (auto-created):

```json
{
  "records": [
    {
      "sha256": "abc123...",
      "contentType": "game-binary",
      "gameId": "pz",
      "gameVersion": "42.16",
      "platform": "windows-x64",
      "sizeBytes": 3200000000,
      "trustLevel": "verified",
      "uploadCount": 3,
      "firstUploadedAt": "2026-06-01T10:00:00Z",
      "lastVerifiedAt": "2026-06-20T08:00:00Z"
    }
  ],
  "submissions": {
    "sha256-hash": ["192.168.1.0/24", "10.0.0.0/24"]
  }
}
```

---

## Storage Management

### Monitor Disk Usage

```bash
du -sh ~/pz-registry/data/*
df -h ~/pz-registry
```

### Cleanup Old Versions (manual)

```bash
# Delete v41.x (old version)
rm -rf ~/pz-registry/data/41*

# Prune registry.json entries
# (manually edit or use admin API when built)
```

---

## Iran-Accessible Deployment

For Iran/restricted regions, use:

- **Hetzner** (Germany): Available from Iran, HTTPS accessible
- **Local ISP VPS**: If you control it
- **Tor/VPN**: Not recommended for public registry (slow)

### Network Test from Iran

```bash
curl -v https://registry.yourdomain.com/api/v1/registry/catalog?game=pz
```

If blocked, check:
- Firewall rules (allow port 443)
- DNS resolution
- ISP filtering (unlikely for HTTPS)

---

## Monitoring

### Health Check

```bash
curl http://localhost:8080/api/v1/health
```

### Logs

```bash
# Docker
docker-compose logs -f registry

# Systemd
journalctl -u pz-registry -f
```

### Prometheus Metrics

```bash
curl http://localhost:8080/metrics
```

Monitor:
- `pz_registry_upload_bytes_total` — upload volume
- `pz_registry_download_total` — download count

---

## Troubleshooting

### "Address already in use"

```bash
lsof -i :8080
kill -9 <PID>
```

### "Permission denied" on data directory

```bash
sudo chown -R pz-registry:pz-registry /opt/pz-registry/data
```

### TLS certificate expired

```bash
sudo certbot renew
docker-compose restart registry
```

### Large upload fails

Increase Nginx buffer:

```nginx
client_max_body_size 10G;
proxy_request_buffering off;
```

---

## Backup & Restore

### Backup Registry Data

```bash
tar czf registry-backup-$(date +%Y%m%d).tar.gz ~/pz-registry/data ~/pz-registry/config/
```

### Restore

```bash
tar xzf registry-backup-*.tar.gz -C ~/
docker-compose restart registry
```

---

## Next Steps

1. **Configure your Launcher to use this registry:**
   - Set `backendUrl` in launcher settings to `https://registry.yourdomain.com`

2. **Upload initial versions:**
   - Community members can POST to `/api/v1/registry/versions/.../submit` and PUT the binary

3. **Monitor trust levels:**
   - First version from 1 uploader: `pending`
   - After 3 independent uploads: `verified`
