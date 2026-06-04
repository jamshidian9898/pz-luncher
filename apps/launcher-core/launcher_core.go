package main

type LauncherCore interface {
	JoinServer(serverID string) error
	ResolveManifest(serverID string) error
	ResolvePackages() error
	DownloadMissing() error
	PrepareProfile() error
	Launch() error
}
