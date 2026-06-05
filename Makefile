.PHONY: contracts test join demo dev-api ui-dev build-ui build-mac build-windows build-linux build-all

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
