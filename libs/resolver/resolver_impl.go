package resolver

import (
	"errors"
	"fmt"

	"pzlauncher/libs/contracts"
)

// Resolve transforms a manifest into a topologically ordered list of ResolvedPackage
// It validates for cycles and returns an error if the dependency graph is cyclic.
func (r *defaultResolver) Resolve(manifest contracts.Manifest) ([]contracts.ResolvedPackage, error) {
	// build node map
	nodes := make(map[string]contracts.ResolvedPackage)
	edges := make(map[string][]string)

	for _, m := range manifest.Mods {
		nodes[m.ID] = contracts.ResolvedPackage{
			ID:           m.ID,
			Version:      m.Version,
			SHA256:       m.SHA256,
			Size:         0,
			ProviderName: "",
			DownloadURL:  m.DownloadURL,
			Dependencies: m.Dependencies,
		}
		// ensure edges entry exists
		edges[m.ID] = append([]string{}, m.Dependencies...)
	}

	// topological sort via DFS
	visited := make(map[string]int) // 0=unvisited,1=visiting,2=done
	var order []contracts.ResolvedPackage

	var visit func(string) error
	visit = func(n string) error {
		state, ok := visited[n]
		if ok && state == 2 {
			return nil
		}
		if state == 1 {
			return fmt.Errorf("cycle detected at package %s", n)
		}
		visited[n] = 1
		for _, dep := range edges[n] {
			if _, exists := nodes[dep]; !exists {
				return fmt.Errorf("unknown dependency %s for package %s", dep, n)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		visited[n] = 2
		order = append(order, nodes[n])
		return nil
	}

	for id := range nodes {
		if visited[id] == 0 {
			if err := visit(id); err != nil {
				return nil, err
			}
		}
	}

	// order currently has dependencies before dependents due to post-order append
	// but because we appended after visiting deps, this order is valid
	if len(order) == 0 {
		return nil, errors.New("no packages resolved from manifest")
	}
	return order, nil
}

type defaultResolver struct{}

// NewDefaultResolver returns a basic resolver implementation
func NewDefaultResolver() *defaultResolver { return &defaultResolver{} }
