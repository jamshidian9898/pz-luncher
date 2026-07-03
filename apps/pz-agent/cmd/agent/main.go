// agent is the PZ platform Agent executable.
//
// Responsibilities:
//  1. Discover mods in a local directory
//  2. Push blobs to the Backend Content Store (idempotent, retried)
//  3. Publish a server manifest to the Backend (only when content changes)
//  4. Send periodic heartbeats (always, even on partial failure)
//
// Reliability model (B3):
//   - All network operations use retry.Policy with exponential backoff.
//   - Heartbeat is independent of blob/manifest sync: a backend restart
//     does not silence the agent.
//   - Manifest is published only when the set of mod SHA256s changes
//     (content diff), reducing spurious writes.
//   - Signal handling: SIGINT/SIGTERM trigger a clean shutdown.
//
// On Windows the agent can also run as a native service:
//
//	pz-agent -service install -server my-server -mods C:\pz\mods
//	pz-agent -service start | stop | uninstall
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	serverID := flag.String("server", "", "server ID (auto-detected if omitted)")
	backendURL := flag.String("backend", "http://localhost:8080", "backend base URL")
	modsDir := flag.String("mods", "", "local mods directory to scan (auto-detected if omitted)")
	gameVersion := flag.String("game-version", "42.8", "game version string")
	interval := flag.Duration("interval", 5*time.Minute, "sync interval (0 = run once and exit)")
	token := flag.String("token", "", "agent auth token (or set PZ_AGENT_TOKEN env var)")
	logFile := flag.String("logfile", "", "append logs to this file instead of stderr")
	service := flag.String("service", "", "Windows service control: install | uninstall | start | stop")
	serviceName := flag.String("service-name", defaultServiceName, "Windows service name (set when running multiple agents on one host)")
	flag.Parse()

	opts := Options{
		ServerID:    *serverID,
		BackendURL:  *backendURL,
		ModsDir:     *modsDir,
		GameVersion: *gameVersion,
		Token:       *token,
		Interval:    *interval,
		LogFile:     *logFile,
		ServiceName: *serviceName,
	}

	if *service != "" {
		if err := handleServiceCommand(*service, opts); err != nil {
			log.Fatal(err)
		}
		return
	}

	if isWindowsService() {
		if err := runWindowsService(opts); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Interactive mode — cancelled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runAgent(ctx, opts); err != nil {
		log.Fatal(err)
	}
}
