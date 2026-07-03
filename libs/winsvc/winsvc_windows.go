//go:build windows

// Package winsvc lets a Go daemon run as a native Windows service without
// external tools (NSSM etc). Used by pz-agent and pz-backend.
package winsvc

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Config describes the service to install.
type Config struct {
	Name        string
	DisplayName string
	Description string
	// Args is the command line baked into the service registration, so the
	// service runs with the same configuration it was installed with.
	Args []string
}

// IsService reports whether the process was started by the service control manager.
func IsService() bool {
	inService, err := svc.IsWindowsService()
	return err == nil && inService
}

// Run hosts fn under the service control manager. fn must block until its
// context is cancelled; returning an error marks the service as failed so
// the recovery actions (restart) kick in.
func Run(name string, fn func(context.Context) error) error {
	return svc.Run(name, &handler{fn: fn})
}

type handler struct {
	fn func(context.Context) error
}

func (h *handler) Execute(_ []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- h.fn(ctx) }()

	status <- svc.Status{State: svc.Running, Accepts: accepted}
	for {
		select {
		case err := <-done:
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

// Command implements the install | uninstall | start | stop service actions.
func Command(cmd string, cfg Config) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager (run as Administrator?): %w", err)
	}
	defer m.Disconnect()

	switch cmd {
	case "install":
		return install(m, cfg)
	case "uninstall":
		s, err := m.OpenService(cfg.Name)
		if err != nil {
			return fmt.Errorf("service %s not installed: %w", cfg.Name, err)
		}
		defer s.Close()
		if err := s.Delete(); err != nil {
			return fmt.Errorf("delete service: %w", err)
		}
		fmt.Printf("service %s uninstalled\n", cfg.Name)
		return nil
	case "start":
		s, err := m.OpenService(cfg.Name)
		if err != nil {
			return fmt.Errorf("open service: %w", err)
		}
		defer s.Close()
		return s.Start()
	case "stop":
		s, err := m.OpenService(cfg.Name)
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

func install(m *mgr.Mgr, cfg Config) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	if s, err := m.OpenService(cfg.Name); err == nil {
		s.Close()
		return fmt.Errorf("service %s already installed", cfg.Name)
	}

	s, err := m.CreateService(cfg.Name, exe, mgr.Config{
		DisplayName: cfg.DisplayName,
		Description: cfg.Description,
		StartType:   mgr.StartAutomatic,
	}, cfg.Args...)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer s.Close()

	// Restart on failure so a transient crash never permanently kills the daemon.
	_ = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 10 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400)

	fmt.Printf("service %s installed\n", cfg.Name)
	return nil
}
