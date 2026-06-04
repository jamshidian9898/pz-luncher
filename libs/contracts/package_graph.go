package contracts

type PackageGraph struct {
	Nodes map[string]ResolvedPackage `json:"nodes"`
	Edges map[string][]string        `json:"edges"`
}
