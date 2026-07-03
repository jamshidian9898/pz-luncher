package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"pzlauncher/libs/winsvc"
)

const defaultServiceName = "PZAgent"

func isWindowsService() bool { return winsvc.IsService() }

// runWindowsService runs the agent under the Windows service control manager.
func runWindowsService(o Options) error {
	if o.LogFile == "" {
		// Services have no console — default to a log next to the executable.
		if exe, err := os.Executable(); err == nil {
			o.LogFile = filepath.Join(filepath.Dir(exe), o.ServiceName+".log")
		}
	}
	return winsvc.Run(o.ServiceName, func(ctx context.Context) error {
		return runAgent(ctx, o)
	})
}

// handleServiceCommand implements -service install|uninstall|start|stop.
func handleServiceCommand(cmd string, o Options) error {
	args := []string{
		"-backend", o.BackendURL,
		"-game-version", o.GameVersion,
		"-interval", o.Interval.String(),
	}
	if o.ServerID != "" {
		args = append(args, "-server", o.ServerID)
	}
	if o.ModsDir != "" {
		args = append(args, "-mods", o.ModsDir)
	}
	if o.Token != "" {
		args = append(args, "-token", o.Token)
	}
	if o.LogFile != "" {
		args = append(args, "-logfile", o.LogFile)
	}
	if o.ServiceName != defaultServiceName {
		args = append(args, "-service-name", o.ServiceName)
	}
	displayName := "PZ Platform Agent"
	if o.ServerID != "" {
		displayName = fmt.Sprintf("PZ Platform Agent (%s)", o.ServerID)
	}
	return winsvc.Command(cmd, winsvc.Config{
		Name:        o.ServiceName,
		DisplayName: displayName,
		Description: "Publishes Project Zomboid server mods and heartbeats to the PZ backend.",
		Args:        args,
	})
}
