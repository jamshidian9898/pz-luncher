package pzagent

type Agent interface {
	Heartbeat() error
	GenerateManifest() error
	DetectMods() error
	Sync() error
}
