// Command-line join flow for testing RFC-0030–0033 without Wails UI.
//
// Two modes:
//   - offline (default): resolve from local registry fixtures (v1 path)
//   - backend: POST /api/v1/join/{serverId} and run the v2 download pipeline,
//     exercising the same path as the launcher UI. Used by lab verify scripts.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"pzlauncher/libs/pipeline"
	"pzlauncher/libs/settings"
)

func main() {
	serverID := flag.String("server", "demo-survival", "server id from examples/servers.json")
	backendURL := flag.String("backend", "", "backend base URL; when set, join via POST /api/v1/join (v2 path)")
	launch := flag.Bool("launch", false, "launch game after join")
	flag.Parse()

	root := pipeline.WorkspaceRoot()
	st, _ := settings.Load(root)
	settings.ApplyGamePathEnv(st)
	cfg := settings.ToPipelineConfig(root, st)
	svc := pipeline.NewService(cfg)

	emit := func(ev pipeline.Event) {
		fmt.Printf("[%s] session=%s pkg=%s", ev.Type, ev.SessionID, ev.PackageID)
		if ev.Error != "" {
			fmt.Printf(" err=%s", ev.Error)
		}
		fmt.Println()
	}

	ctx := context.Background()

	if *backendURL != "" {
		jr, err := backendJoin(ctx, *backendURL, *serverID)
		if err != nil {
			log.Fatal(err)
		}
		result, err := svc.RunJoinFromBackend(ctx, *jr, emit)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("OK profile=%s mods=%d\n", result.ProfilePath, len(jr.DownloadPlan))
		if *launch {
			if err := svc.LaunchFromBackend(ctx, *serverID, result.ProfilePath, *jr, emit); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Launch OK")
		}
		os.Exit(0)
	}

	result, err := svc.RunJoin(ctx, *serverID, emit)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("OK profile=%s mods=%d\n", result.ProfilePath, len(result.Plan.OrderedMods))

	if *launch {
		if err := svc.Launch(ctx, *serverID, result.ProfilePath, emit); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Launch OK")
	}
	os.Exit(0)
}

// backendJoin calls POST /api/v1/join/{serverId} — the same call the
// launcher UI makes — and decodes the JoinResponse.
func backendJoin(ctx context.Context, baseURL, serverID string) (*pipeline.BackendJoinResponse, error) {
	url := fmt.Sprintf("%s/api/v1/join/%s", baseURL, serverID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("backend join: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var envelope struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&envelope)
		return nil, fmt.Errorf("backend join: %s %s: %s", resp.Status, envelope.Error.Code, envelope.Error.Message)
	}
	var jr pipeline.BackendJoinResponse
	if err := json.NewDecoder(resp.Body).Decode(&jr); err != nil {
		return nil, fmt.Errorf("backend join: decode: %w", err)
	}
	return &jr, nil
}
