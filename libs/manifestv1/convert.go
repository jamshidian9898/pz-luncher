package manifestv1

import (
	"fmt"
	"strconv"

	"pzlauncher/libs/contracts"
)

// ToLegacyManifest maps RFC-0030 to the existing contracts.Manifest type.
func ToLegacyManifest(m *ServerManifest) contracts.Manifest {
	ver, _ := strconv.Atoi(m.Version)
	if ver == 0 {
		ver = 1
	}
	mods := make([]contracts.ManifestMod, len(m.Mods))
	for i, mod := range m.Mods {
		mods[i] = contracts.ManifestMod{
			ID:           mod.ID,
			Version:      mod.Version,
			SHA256:       mod.SHA256,
			DownloadURL:  mod.DownloadURL,
			Dependencies: append([]string(nil), mod.Dependencies...),
		}
	}
	return contracts.Manifest{
		ID:          fmt.Sprintf("%s-v%s", m.ServerID, m.Version),
		ServerID:    m.ServerID,
		Version:     ver,
		GameVersion: m.GameVersion,
		Mods:        mods,
	}
}
