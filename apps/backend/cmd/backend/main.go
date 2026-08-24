// backend is the v2.0.0 Platform control plane.
// Run: go run ./apps/backend/cmd/backend
//
// On Windows the backend can also run as a native service:
//
//	pz-backend -service install -addr :8080 -registry C:\pz\registry.json ...
//	pz-backend -service start | stop | uninstall
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"pzlauncher/apps/backend/internal/api"
	"pzlauncher/apps/backend/internal/auth"
	"pzlauncher/apps/backend/internal/content"
	"pzlauncher/apps/backend/internal/manifest"
	"pzlauncher/apps/backend/internal/obs"
	"pzlauncher/apps/backend/internal/registry"
	"pzlauncher/apps/backend/internal/storage"
	"pzlauncher/libs/winsvc"
)

const serviceName = "PZBackend"

type options struct {
	Addr         string
	RegistryFile string
	StoreDir     string
	ContentFile  string
	FixturesDir  string
	DeployDir    string
	NoAuth       bool
	LogFile      string
	TokensFile   string
	ManifestsDir string
	AdminToken   string
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	registryFile := flag.String("registry", "apps/backend/registry.json", "path to registry.json")
	storeDir := flag.String("store", "apps/backend/store", "content-addressable blob store directory")
	contentFile := flag.String("content-registry", "apps/backend/content-registry.json", "path to content registry metadata file (RFC-0059)")
	fixturesDir := flag.String("fixtures", "fixtures", "fixtures root for demo blob seeding")
	deployDir := flag.String("deploy", "", "directory to serve under /releases/ and /install-agent.sh (optional)")
	noAuth := flag.Bool("no-auth", false, "disable agent token auth (dev/test only)")
	logFile := flag.String("logfile", "", "append logs to this file instead of stderr")
	tokensFile := flag.String("tokens", "apps/backend/agent-tokens.json", "path to persist agent access tokens (survives restarts)")
	manifestsDir := flag.String("manifests-dir", "apps/backend/manifests", "directory to persist versioned server manifests (survives restarts)")
	adminToken := flag.String("admin-token", os.Getenv("PZ_ADMIN_TOKEN"), "operator secret enabling /api/v1/admin/* (also PZ_ADMIN_TOKEN env)")
	service := flag.String("service", "", "Windows service control: install | uninstall | start | stop")
	flag.Parse()

	opts := options{
		Addr:         *addr,
		RegistryFile: *registryFile,
		StoreDir:     *storeDir,
		ContentFile:  *contentFile,
		FixturesDir:  *fixturesDir,
		DeployDir:    *deployDir,
		NoAuth:       *noAuth,
		LogFile:      *logFile,
		TokensFile:   *tokensFile,
		ManifestsDir: *manifestsDir,
		AdminToken:   *adminToken,
	}

	if *service != "" {
		if err := winsvc.Command(*service, serviceConfig(opts)); err != nil {
			log.Fatal(err)
		}
		return
	}

	if winsvc.IsService() {
		if opts.LogFile == "" {
			if exe, err := os.Executable(); err == nil {
				opts.LogFile = filepath.Join(filepath.Dir(exe), "pz-backend.log")
			}
		}
		if err := winsvc.Run(serviceName, func(ctx context.Context) error {
			return run(ctx, opts)
		}); err != nil {
			log.Fatal(err)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, opts); err != nil {
		log.Fatal(err)
	}
}

// run starts the backend and blocks until ctx is cancelled or the server fails.
func run(ctx context.Context, o options) error {
	if o.LogFile != "" {
		f, err := os.OpenFile(o.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("backend: open log file: %w", err)
		}
		defer f.Close()
		log.SetOutput(f)
		os.Stderr = f // obs logging writes to stderr
	}

	reg, err := registry.LoadFromFile(o.RegistryFile)
	if err != nil {
		obs.LogError(ctx, "registry.load_failed",
			"path", o.RegistryFile, "error", err,
			"msg", "starting with empty registry",
		)
		reg = registry.NewMemoryRegistry()
	}

	manifests, err := manifest.NewDiskStore(o.ManifestsDir)
	if err != nil {
		obs.LogError(ctx, "manifests.load_failed", "path", o.ManifestsDir, "error", err)
		return fmt.Errorf("manifests: %w", err)
	}
	reg.SetManifestStore(manifests)

	baseURL := addrToBaseURL(o.Addr)

	store, err := storage.NewDiskStore(o.StoreDir)
	if err != nil {
		obs.LogError(ctx, "storage.init_failed", "error", err)
		return fmt.Errorf("storage: %w", err)
	}
	seedDemoBlobs(store, o.FixturesDir)

	var tokens *auth.Store
	if !o.NoAuth {
		tokens, err = auth.NewPersistentStore(o.TokensFile)
		if err != nil {
			obs.LogError(ctx, "tokens.load_failed", "path", o.TokensFile, "error", err)
			return fmt.Errorf("tokens: %w", err)
		}
	}

	contentReg, err := content.NewRegistry(o.ContentFile, content.DefaultThreshold)
	if err != nil {
		obs.LogError(ctx, "content_registry.load_failed",
			"path", o.ContentFile, "error", err,
			"msg", "starting with empty content registry",
		)
		contentReg, _ = content.NewRegistry("", content.DefaultThreshold)
	}

	mux := api.NewRouter(reg, baseURL, store, tokens, o.AdminToken, contentReg)

	// Serve deploy assets if -deploy is set.
	// GET /install-agent.sh  → deploy/install-agent.sh
	// GET /releases/*         → deploy/releases/*
	if o.DeployDir != "" {
		if _, err := os.Stat(o.DeployDir); err == nil {
			http.Handle("/install-agent.sh", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/x-shellscript")
				http.ServeFile(w, r, o.DeployDir+"/install-agent.sh")
			}))
			http.Handle("/releases/", http.StripPrefix("/releases/",
				http.FileServer(http.Dir(o.DeployDir+"/releases"))))
			obs.Log(ctx, "backend.deploy_dir", "path", o.DeployDir)
		}
	}

	authMode := "enabled"
	if o.NoAuth {
		authMode = "disabled (dev)"
	}
	obs.Log(ctx, "backend.start",
		"addr", o.Addr,
		"base_url", baseURL,
		"store", o.StoreDir,
		"auth", authMode,
	)

	srv := &http.Server{Addr: o.Addr, Handler: mux}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return nil
	}
}

func serviceConfig(o options) winsvc.Config {
	args := []string{
		"-addr", o.Addr,
		"-registry", o.RegistryFile,
		"-store", o.StoreDir,
		"-content-registry", o.ContentFile,
		"-fixtures", o.FixturesDir,
		"-tokens", o.TokensFile,
		"-manifests-dir", o.ManifestsDir,
	}
	if o.DeployDir != "" {
		args = append(args, "-deploy", o.DeployDir)
	}
	if o.NoAuth {
		args = append(args, "-no-auth")
	}
	if o.LogFile != "" {
		args = append(args, "-logfile", o.LogFile)
	}
	return winsvc.Config{
		Name:        serviceName,
		DisplayName: "PZ Platform Backend",
		Description: "PZ platform control plane: server registry, content store, join API.",
		Args:        args,
	}
}

func addrToBaseURL(addr string) string {
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "localhost" + host
	}
	return fmt.Sprintf("http://%s", host)
}
