package providers

import (
	"context"
	"log"
	"time"

	"pzlauncher/libs/contracts"
)

// ApplyFallback inspects the provided packages and a prioritized list of
// providers. It sets `ProviderName` and `Cached` on each package.
// It returns detailed decision traces showing:
//   - Which providers were checked and in what order
//   - Results (exists/miss/errors) with timing
//   - Final decision with complete reasoning
//   - Fallback chain visualization
func ApplyFallback(ctx context.Context, pkgs []contracts.ResolvedPackage, provs []Provider) ([]contracts.ResolvedPackage, []contracts.ProviderDecision, error) {
	var decisions []contracts.ProviderDecision

	for i := range pkgs {
		p := &pkgs[i]
		startTime := time.Now()

		decision := contracts.ProviderDecision{
			PackageID:      p.ID,
			PackageVersion: p.Version,
			PackageSHA256:  p.SHA256,
			DecisionAt:     startTime,
			FallbackChain:  extractProviderNames(provs),
		}

		chosen := ""
		finalReason := ""

		for _, pr := range provs {
			attemptStart := time.Now()

			exists, err := pr.Exists(ctx, contracts.PackageMetadata{ID: p.ID, Version: p.Version, SHA256: p.SHA256})
			duration := time.Since(attemptStart).Milliseconds()

			attempt := contracts.ProviderAttempt{
				ProviderName: pr.Name(),
				CheckedAt:    attemptStart,
				Exists:       exists,
				DurationMs:   duration,
			}

			if err != nil {
				attempt.Error = err.Error()
				log.Printf("[TRACE] provider %s check error for %s: %v", pr.Name(), p.ID, err)
			} else if exists {
				// Add cache path for local providers
				if lcp, ok := pr.(*LocalCacheProvider); ok {
					attempt.CachePath = lcp.CacheDir + "/" + p.SHA256
				}
			}

			decision.Attempts = append(decision.Attempts, attempt)

			if err == nil && exists && chosen == "" {
				p.ProviderName = pr.Name()
				p.Cached = true
				chosen = pr.Name()
				finalReason = "found in " + pr.Name()
				log.Printf("[TRACE] %s → chosen %s (cached)", p.ID, pr.Name())
			}
		}

		if chosen == "" {
			// No provider had it locally — pick fallback
			p.Cached = false
			if len(provs) > 1 {
				p.ProviderName = provs[1].Name()
				finalReason = "cache miss → fallback to " + provs[1].Name()
			} else if len(provs) > 0 {
				p.ProviderName = provs[0].Name()
				finalReason = "only provider available: " + provs[0].Name()
			} else {
				p.ProviderName = "none"
				finalReason = "no providers available"
			}
			log.Printf("[TRACE] %s → fallback to %s (not cached)", p.ID, p.ProviderName)
		}

		decision.ChosenProvider = p.ProviderName
		decision.Cached = p.Cached
		decision.FinalReason = finalReason
		decision.TotalDurationMs = time.Since(startTime).Milliseconds()

		decisions = append(decisions, decision)
	}

	return pkgs, decisions, nil
}

func extractProviderNames(provs []Provider) []string {
	names := make([]string, len(provs))
	for i, p := range provs {
		names[i] = p.Name()
	}
	return names
}
