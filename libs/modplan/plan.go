package modplan

import "pzlauncher/libs/manifestv1"

type ResolvedMod struct {
	ID           string
	Name         string
	Version      string
	SHA256       string
	SizeBytes    int64
	WorkshopID   string
	DownloadURL  string
	Depth        int
	DependsOn    []string
}

type ResolverWarning struct {
	Code    string
	Message string
	ModID   string
}

type ResolvedModPlan struct {
	ServerID        string
	ManifestVersion string
	GameVersion     string
	OrderedMods     []ResolvedMod
	Skipped         []string
	Warnings        []ResolverWarning
}

func (p *ResolvedModPlan) ModIDs() []string {
	ids := make([]string, len(p.OrderedMods))
	for i, m := range p.OrderedMods {
		ids[i] = m.ID
	}
	return ids
}

// FromManifest resolves dependencies for a validated ServerManifest.
func FromManifest(m *manifestv1.ServerManifest) (*ResolvedModPlan, error) {
	return Resolve(ResolveInput{Manifest: m})
}

type ResolveInput struct {
	Manifest      *manifestv1.ServerManifest
	InstalledMods map[string]string // id -> version
}
