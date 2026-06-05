// backend is the v2.0.0 Platform control plane.
// Run: go run ./apps/backend/cmd/backend
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"pzlauncher/apps/backend/internal/api"
	"pzlauncher/apps/backend/internal/registry"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	registryFile := flag.String("registry", "apps/backend/registry.json", "path to registry.json")
	flag.Parse()

	reg, err := registry.LoadFromFile(*registryFile)
	if err != nil {
		log.Printf("warn: could not load registry file %q: %v — starting with empty registry", *registryFile, err)
		reg = registry.NewMemoryRegistry()
	}

	baseURL := addrToBaseURL(*addr)
	mux := api.NewRouter(reg, baseURL)

	log.Printf("backend listening on %s (baseURL: %s)", *addr, baseURL)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func addrToBaseURL(addr string) string {
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "localhost" + host
	}
	return fmt.Sprintf("http://%s", host)
}
