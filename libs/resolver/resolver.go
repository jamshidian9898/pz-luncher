package resolver

import "pzlauncher/libs/contracts"

type Resolver interface {
	Resolve(manifest contracts.Manifest) ([]contracts.ResolvedPackage, error)
}
