package modplan

import (
	"fmt"
	"strings"

	"pzlauncher/libs/manifestv1"
)

func Resolve(in ResolveInput) (*ResolvedModPlan, error) {
	m := in.Manifest
	if m == nil || len(m.Mods) == 0 {
		return nil, fmt.Errorf("RESOLVER_EMPTY_MANIFEST: no mods in manifest")
	}

	nodes := make(map[string]manifestv1.ModEntry, len(m.Mods))
	edges := make(map[string][]string, len(m.Mods))
	for _, mod := range m.Mods {
		if mod.Optional {
			continue
		}
		nodes[mod.ID] = mod
		edges[mod.ID] = append([]string{}, mod.Dependencies...)
	}

	visited := make(map[string]int)
	var order []ResolvedMod
	depth := make(map[string]int)

	var visit func(string, int) error
	visit = func(id string, d int) error {
		state := visited[id]
		if state == 2 {
			return nil
		}
		if state == 1 {
			return fmt.Errorf("RESOLVER_CYCLE: cycle detected at %s", id)
		}
		visited[id] = 1
		if d > depth[id] {
			depth[id] = d
		}
		for _, dep := range edges[id] {
			if _, ok := nodes[dep]; !ok {
				return fmt.Errorf("RESOLVER_UNKNOWN_DEP: unknown dependency %q for %s", dep, id)
			}
			if err := visit(dep, d+1); err != nil {
				return err
			}
		}
		visited[id] = 2
		mod := nodes[id]
		if in.InstalledMods != nil {
			if got, ok := in.InstalledMods[mod.ID]; ok && got != "" && got != mod.Version {
				return fmt.Errorf("RESOLVER_VERSION_CONFLICT: %s installed %s need %s", mod.ID, got, mod.Version)
			}
		}
		if strings.TrimSpace(mod.WorkshopID) == "" && strings.TrimSpace(mod.DownloadURL) == "" {
			return fmt.Errorf("RESOLVER_MISSING_SOURCE: mod %s has no workshopId or downloadUrl", mod.ID)
		}
		order = append(order, ResolvedMod{
			ID:          mod.ID,
			Name:        mod.Name,
			Version:     mod.Version,
			SHA256:      mod.SHA256,
			SizeBytes:   mod.SizeBytes,
			WorkshopID:  mod.WorkshopID,
			DownloadURL: mod.DownloadURL,
			Depth:       depth[id],
			DependsOn:   append([]string{}, mod.Dependencies...),
		})
		return nil
	}

	for id := range nodes {
		if visited[id] == 0 {
			if err := visit(id, 0); err != nil {
				return nil, err
			}
		}
	}

	if len(m.Profile.ModLoadOrder) > 0 {
		order = applyLoadOrder(order, m.Profile.ModLoadOrder)
	}

	return &ResolvedModPlan{
		ServerID:        m.ServerID,
		ManifestVersion: m.Version,
		GameVersion:     m.GameVersion,
		OrderedMods:     order,
	}, nil
}

func applyLoadOrder(mods []ResolvedMod, loadOrder []string) []ResolvedMod {
	index := make(map[string]int, len(loadOrder))
	for i, id := range loadOrder {
		index[id] = i
	}
	// stable sort: known ids first by load order, then remainder in topo order
	out := make([]ResolvedMod, 0, len(mods))
	used := make(map[string]bool)
	for _, id := range loadOrder {
		for _, m := range mods {
			if m.ID == id {
				out = append(out, m)
				used[id] = true
				break
			}
		}
	}
	for _, m := range mods {
		if !used[m.ID] {
			out = append(out, m)
		}
	}
	return out
}
