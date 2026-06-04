package game

import "pzlauncher/libs/contracts"

type InstallationFinder interface {
	FindInstallation() (contracts.GameInstallation, error)
}

// GameLauncher is responsible for launching an installed copy of Project Zomboid.
type GameLauncher interface {
	Launch(installation contracts.GameInstallation, request contracts.LaunchRequest) (contracts.LaunchResult, error)
}
