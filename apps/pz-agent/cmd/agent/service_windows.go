//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "PZAgent"

// isWindowsService reports whether we were started by the service control manager.
func isWindowsService() bool {
	inService, err := svc.IsWindowsService()
	return err == nil && inService
}

// runWindowsService runs the agent under the Windows service control manager.
func runWindowsService(o Options) error {
	if o.LogFile == "" {
		// Services have no console — default to a log next to the executable.
		if exe, err := os.Executable(); err == nil {
			o.LogFile = filepath.Join(filepath.Dir(exe), "pz-agent.log")
		}
	}
	return svc.Run(serviceName, &agentService{opts: o})
}

type agentService struct {
	opts Options
}

func (s *agentService) Execute(_ []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- runAgent(ctx, s.opts) }()

	status <- svc.Status{State: svc.Running, Accepts: accepted}
	for {
		select {
		case err := <-done:
			// Run-once mode (Interval == 0) or fatal config error.
			if err != nil {
				return true, 1
			}
			return false, 0
		case c := <-req:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				cancel()
				select {
				case <-done:
				case <-time.After(15 * time.Second):
				}
				return false, 0
			}
		}
	}
}

// handleServiceCommand implements -service install|uninstall|start|stop.
func handleServiceCommand(cmd string, o Options) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager (run as Administrator?): %w", err)
	}
	defer m.Disconnect()

	switch cmd {
	case "install":
		return installService(m, o)
	case "uninstall":
		return uninstallService(m)
	case "start":
		s, err := m.OpenService(serviceName)
		if err != nil {
			return fmt.Errorf("open service: %w", err)
		}
		defer s.Close()
		return s.Start()
	case "stop":
		s, err := m.OpenService(serviceName)
		if err != nil {
			return fmt.Errorf("open service: %w", err)
		}
		defer s.Close()
		_, err = s.Control(svc.Stop)
		return err
	default:
		return fmt.Errorf("unknown -service command %q (want install|uninstall|start|stop)", cmd)
	}
}

func installService(m *mgr.Mgr, o Options) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	if s, err := m.OpenService(serviceName); err == nil {
		s.Close()
		return fmt.Errorf("service %s already installed", serviceName)
	}

	// Bake the current flags into the service command line so the service
	// runs with the same configuration it was installed with.
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

	s, err := m.CreateService(serviceName, exe, mgr.Config{
		DisplayName: "PZ Platform Agent",
		Description: "Publishes Project Zomboid server mods and heartbeats to the PZ backend.",
		StartType:   mgr.StartAutomatic,
	}, args...)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer s.Close()

	// Restart on failure so a backend outage never permanently kills the agent.
	_ = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 10 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400)

	fmt.Printf("service %s installed (start with: pz-agent -service start)\n", serviceName)
	return nil
}

func uninstallService(m *mgr.Mgr) error {
	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s not installed: %w", serviceName, err)
	}
	defer s.Close()
	if err := s.Delete(); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	fmt.Printf("service %s uninstalled\n", serviceName)
	return nil
}
