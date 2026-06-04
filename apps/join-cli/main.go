// Command-line join flow for testing RFC-0030–0033 without Wails UI.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"pzlauncher/libs/pipeline"
	"pzlauncher/libs/settings"
)

func main() {
	serverID := flag.String("server", "demo-survival", "server id from examples/servers.json")
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

	result, err := svc.RunJoin(context.Background(), *serverID, emit)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("OK profile=%s mods=%d\n", result.ProfilePath, len(result.Plan.OrderedMods))

	if *launch {
		if err := svc.Launch(context.Background(), *serverID, result.ProfilePath, emit); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Launch OK")
	}
	os.Exit(0)
}
