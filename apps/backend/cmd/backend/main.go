// backend is the v2.0.0 Platform control plane.
// Run: go run ./apps/backend/cmd/backend
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"pzlauncher/apps/backend/internal/api"
	"pzlauncher/apps/backend/internal/auth"
	"pzlauncher/apps/backend/internal/obs"
	"pzlauncher/apps/backend/internal/registry"
	"pzlauncher/apps/backend/internal/storage"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	registryFile := flag.String("registry", "apps/backend/registry.json", "path to registry.json")
	storeDir := flag.String("store", "apps/backend/store", "content-addressable blob store directory")
	fixturesDir := flag.String("fixtures", "fixtures", "fixtures root for demo blob seeding")
	deployDir := flag.String("deploy", "", "directory to serve under /releases/ and /install-agent.sh (optional)")
	noAuth := flag.Bool("no-auth", false, "disable agent token auth (dev/test only)")
	flag.Parse()

	reg, err := registry.LoadFromFile(*registryFile)
	if err != nil {
		obs.LogError(context.Background(), "registry.load_failed",
			"path", *registryFile, "error", err,
			"msg", "starting with empty registry",
		)
		reg = registry.NewMemoryRegistry()
	}

	baseURL := addrToBaseURL(*addr)

	store, err := storage.NewDiskStore(*storeDir)
	if err != nil {
		obs.LogError(context.Background(), "storage.init_failed", "error", err)
		log.Fatalf("storage: %v", err)
	}
	seedDemoBlobs(store, *fixturesDir)

	var tokens *auth.Store
	if !*noAuth {
		tokens = auth.NewStore()
	}

	mux := api.NewRouter(reg, baseURL, store, tokens)

	// Serve deploy assets if -deploy is set.
	// GET /install-agent.sh  → deploy/install-agent.sh
	// GET /releases/*         → deploy/releases/*
	if *deployDir != "" {
		if _, err := os.Stat(*deployDir); err == nil {
			http.Handle("/install-agent.sh", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/x-shellscript")
				http.ServeFile(w, r, *deployDir+"/install-agent.sh")
			}))
			http.Handle("/releases/", http.StripPrefix("/releases/",
				http.FileServer(http.Dir(*deployDir+"/releases"))))
			obs.Log(context.Background(), "backend.deploy_dir", "path", *deployDir)
		}
	}

	authMode := "enabled"
	if *noAuth {
		authMode = "disabled (dev)"
	}
	obs.Log(context.Background(), "backend.start",
		"addr", *addr,
		"base_url", baseURL,
		"store", *storeDir,
		"auth", authMode,
	)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func addrToBaseURL(addr string) string {
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "localhost" + host
	}
	return fmt.Sprintf("http://%s", host)
}
