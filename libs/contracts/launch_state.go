package contracts

type LaunchState string

const (
	LaunchStateIdle              LaunchState = "Idle"
	LaunchStateResolvingManifest LaunchState = "ResolvingManifest"
	LaunchStateResolvingPackages LaunchState = "ResolvingPackages"
	LaunchStateCreatingSession   LaunchState = "CreatingSession" // NEW: Session initialization
	LaunchStateDownloading       LaunchState = "Downloading"     // Execution of provider decisions
	LaunchStateVerifying         LaunchState = "Verifying"       // NEW: Integrity checks post-download
	LaunchStateMaterializing     LaunchState = "Materializing"   // NEW: Profile assembly
	LaunchStateBuildingProfile   LaunchState = "BuildingProfile"
	LaunchStateLaunching         LaunchState = "Launching"
	LaunchStateRunning           LaunchState = "Running"
	LaunchStateStopped           LaunchState = "Stopped"
)
