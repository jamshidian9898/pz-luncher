.PHONY: contracts test join demo dev-api backend agent ui-dev build-ui \
        build-mac build-mac-intel build-windows build-linux build-all \
        release-backend release-agent release docker-backend docker-agent docker-all \
        test-stack test-stack-down test-stack-logs test-stack-status

APP_DIR   = apps/launcher-ui
DIST_DIR  = dist
APP_NAME  = pz-launcher
VERSION  ?= beta-1

contracts:
	go run ./tools/gencontracts

test:
	go test ./libs/...

join:
	go run ./apps/join-cli -server=demo-survival

demo: join
	go run ./apps/join-cli -server=demo-survival -launch

dev-api:
	go run ./apps/dev-api

backend:
	go run ./apps/backend/cmd/backend

agent:
	go run ./apps/pz-agent/cmd/agent

ui-dev:
	cd $(APP_DIR)/frontend && npm run dev

build-ui:
	cd $(APP_DIR)/frontend && npm ci && npm run build

build-mac: build-ui
	mkdir -p $(DIST_DIR)
	cd $(APP_DIR) && \
		CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		go build -tags desktop -ldflags="-s -w -X main.Version=$(VERSION)" \
		-o ../../$(DIST_DIR)/$(APP_NAME)-mac-arm64 .
	@echo "✓ macOS arm64 → $(DIST_DIR)/$(APP_NAME)-mac-arm64"

build-mac-intel: build-ui
	mkdir -p $(DIST_DIR)
	cd $(APP_DIR) && \
		CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		go build -tags desktop -ldflags="-s -w -X main.Version=$(VERSION)" \
		-o ../../$(DIST_DIR)/$(APP_NAME)-mac-amd64 .
	@echo "✓ macOS amd64 → $(DIST_DIR)/$(APP_NAME)-mac-amd64"

build-windows: build-ui
	mkdir -p $(DIST_DIR)
	cd $(APP_DIR) && \
		CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
		go build -tags desktop -ldflags="-s -w -H windowsgui -X main.Version=$(VERSION)" \
		-o ../../$(DIST_DIR)/$(APP_NAME)-windows-amd64.exe .
	@echo "✓ Windows amd64 → $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe"

build-linux: build-ui
	mkdir -p $(DIST_DIR)
	cd $(APP_DIR) && \
		CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build -tags desktop -ldflags="-s -w -X main.Version=$(VERSION)" \
		-o ../../$(DIST_DIR)/$(APP_NAME)-linux-amd64 .
	@echo "✓ Linux amd64 → $(DIST_DIR)/$(APP_NAME)-linux-amd64"

build-all: build-mac build-windows build-linux
	@echo ""
	@echo "All builds complete:"
	@ls -lh $(DIST_DIR)/

# --- Release targets (server binaries: linux-amd64 + linux-arm64) ---

release-backend:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build -ldflags="-s -w -X main.Version=$(VERSION)" \
		-o $(DIST_DIR)/pz-backend-linux-amd64 ./apps/backend/cmd/backend
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
		go build -ldflags="-s -w -X main.Version=$(VERSION)" \
		-o $(DIST_DIR)/pz-backend-linux-arm64 ./apps/backend/cmd/backend
	@echo "✓ Backend → $(DIST_DIR)/pz-backend-linux-{amd64,arm64}"

release-agent:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build -ldflags="-s -w -X main.Version=$(VERSION)" \
		-o $(DIST_DIR)/pz-agent-linux-amd64 ./apps/pz-agent/cmd/agent
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
		go build -ldflags="-s -w -X main.Version=$(VERSION)" \
		-o $(DIST_DIR)/pz-agent-linux-arm64 ./apps/pz-agent/cmd/agent
	@echo "✓ Agent → $(DIST_DIR)/pz-agent-linux-{amd64,arm64}"

release: release-backend release-agent build-mac build-windows build-linux
	@echo ""
	@echo "Release $(VERSION) complete:"
	@ls -lh $(DIST_DIR)/

# --- Docker targets ---

docker-backend:
	docker build -f apps/backend/Dockerfile -t pz-backend:$(VERSION) -t pz-backend:latest .
	@echo "✓ Docker image: pz-backend:$(VERSION)"

docker-agent:
	docker build -f apps/pz-agent/Dockerfile -t pz-agent:$(VERSION) -t pz-agent:latest .
	@echo "✓ Docker image: pz-agent:$(VERSION)"

docker-all: docker-backend docker-agent
	@echo "✓ All Docker images built."

# --- Test Stack (full local environment) ---

test-stack:
	@echo "=== Starting PZ Test Stack ==="
	@echo "Backend:    http://localhost:8080"
	@echo "Grafana:    http://localhost:3000  (admin/admin)"
	@echo "Prometheus: http://localhost:9090"
	@echo "Loki:       http://localhost:3100"
	@echo ""
	docker compose -f docker-compose.test.yml up --build -d
	@echo ""
	@echo "✓ Stack is up. Run 'make test-stack-logs' to follow logs."

test-stack-down:
	docker compose -f docker-compose.test.yml down -v
	@echo "✓ Stack stopped and volumes removed."

test-stack-logs:
	docker compose -f docker-compose.test.yml logs -f --tail=50

test-stack-status:
	@echo "=== Container Status ==="
	@docker compose -f docker-compose.test.yml ps
	@echo ""
	@echo "=== Backend Servers ==="
	@curl -s http://localhost:8080/api/v1/servers | python3 -m json.tool 2>/dev/null || echo "(backend not ready)"
	@echo ""
	@echo "=== Agent Status ==="
	@curl -s http://localhost:8080/api/v1/agents | python3 -m json.tool 2>/dev/null || echo "(backend not ready)"
