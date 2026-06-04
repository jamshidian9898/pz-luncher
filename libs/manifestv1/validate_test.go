package manifestv1

import "testing"

func TestValidate_ok(t *testing.T) {
	m := &ServerManifest{
		ServerID:    "s1",
		Version:     "1",
		GameVersion: "42.8",
		Mods: []ModEntry{{
			ID: "a", Name: "A", Version: "1", SHA256: "4fbc716b086d0746df8f5c7c04064ef6b0ef30ec8c2b9ea6ff9d7c8222fcb0fd",
			WorkshopID: "123", Dependencies: []string{},
		}},
	}
	if err := Validate(m); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_unknownDep(t *testing.T) {
	m := &ServerManifest{
		ServerID: "s1", Version: "1", GameVersion: "42.8",
		Mods: []ModEntry{
			{ID: "a", Version: "1", SHA256: "4fbc716b086d0746df8f5c7c04064ef6b0ef30ec8c2b9ea6ff9d7c8222fcb0fd", WorkshopID: "1", Dependencies: []string{"missing"}},
		},
	}
	if err := Validate(m); err == nil {
		t.Fatal("expected MANIFEST_UNKNOWN_DEP")
	}
}
