//go:build !windows

package main

import "fmt"

func isWindowsService() bool { return false }

func runWindowsService(Options) error {
	return fmt.Errorf("windows service mode is only available on Windows")
}

func handleServiceCommand(string, Options) error {
	return fmt.Errorf("-service is only available on Windows; use deploy/install-agent.sh (systemd) on Linux")
}
