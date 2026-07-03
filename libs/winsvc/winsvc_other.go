//go:build !windows

package winsvc

import (
	"context"
	"fmt"
)

// Config describes the service to install. See winsvc_windows.go.
type Config struct {
	Name        string
	DisplayName string
	Description string
	Args        []string
}

func IsService() bool { return false }

func Run(string, func(context.Context) error) error {
	return fmt.Errorf("winsvc: service mode is only available on Windows")
}

func Command(string, Config) error {
	return fmt.Errorf("winsvc: -service is only available on Windows")
}
