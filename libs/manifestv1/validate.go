package manifestv1

import (
	"fmt"
	"regexp"
	"strings"
)

var sha256Hex = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

type ValidationError struct {
	Code    string
	Message string
	Field   string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func Validate(m *ServerManifest) error {
	if m == nil {
		return &ValidationError{Code: "MANIFEST_INVALID_META", Message: "manifest is nil"}
	}
	if strings.TrimSpace(m.ServerID) == "" {
		return &ValidationError{Code: "MANIFEST_INVALID_META", Message: "serverId required", Field: "serverId"}
	}
	if strings.TrimSpace(m.Version) == "" {
		return &ValidationError{Code: "MANIFEST_INVALID_META", Message: "version required", Field: "version"}
	}
	if strings.TrimSpace(m.GameVersion) == "" {
		return &ValidationError{Code: "MANIFEST_INVALID_META", Message: "gameVersion required", Field: "gameVersion"}
	}

	seen := make(map[string]struct{}, len(m.Mods))
	for i, mod := range m.Mods {
		prefix := fmt.Sprintf("mods[%d]", i)
		if strings.TrimSpace(mod.ID) == "" {
			return &ValidationError{Code: "MANIFEST_INVALID_MOD", Message: "id required", Field: prefix + ".id"}
		}
		if _, dup := seen[mod.ID]; dup {
			return &ValidationError{Code: "MANIFEST_DUPLICATE_MOD", Message: "duplicate mod id", Field: mod.ID}
		}
		seen[mod.ID] = struct{}{}

		if strings.TrimSpace(mod.Version) == "" {
			return &ValidationError{Code: "MANIFEST_INVALID_MOD", Message: "version required", Field: prefix + ".version"}
		}
		if !sha256Hex.MatchString(mod.SHA256) {
			return &ValidationError{Code: "MANIFEST_INVALID_HASH", Message: "sha256 must be 64 hex chars", Field: mod.ID}
		}
		if strings.TrimSpace(mod.WorkshopID) == "" && strings.TrimSpace(mod.DownloadURL) == "" {
			return &ValidationError{Code: "MANIFEST_INVALID_MOD", Message: "workshopId or downloadUrl required", Field: mod.ID}
		}
		for _, dep := range mod.Dependencies {
			if strings.TrimSpace(dep) == "" {
				return &ValidationError{Code: "MANIFEST_UNKNOWN_DEP", Message: "empty dependency id", Field: mod.ID}
			}
		}
	}

	for _, mod := range m.Mods {
		for _, dep := range mod.Dependencies {
			if _, ok := seen[dep]; !ok {
				return &ValidationError{Code: "MANIFEST_UNKNOWN_DEP", Message: fmt.Sprintf("unknown dependency %q", dep), Field: mod.ID}
			}
		}
	}

	return nil
}
