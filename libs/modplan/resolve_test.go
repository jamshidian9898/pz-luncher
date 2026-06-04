package modplan

import (
	"testing"

	"pzlauncher/libs/manifestv1"
)

func TestResolve_order(t *testing.T) {
	m := &manifestv1.ServerManifest{
		ServerID: "s1", Version: "1", GameVersion: "42.8",
		Mods: []manifestv1.ModEntry{
			{ID: "b", Name: "B", Version: "1", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", WorkshopID: "2", Dependencies: []string{"a"}},
			{ID: "a", Name: "A", Version: "1", SHA256: "4fbc716b086d0746df8f5c7c04064ef6b0ef30ec8c2b9ea6ff9d7c8222fcb0fd", WorkshopID: "1", Dependencies: []string{}},
		},
	}
	plan, err := FromManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.OrderedMods) != 2 {
		t.Fatalf("want 2 mods got %d", len(plan.OrderedMods))
	}
	if plan.OrderedMods[0].ID != "a" {
		t.Fatalf("want a first got %s", plan.OrderedMods[0].ID)
	}
}

func TestResolve_cycle(t *testing.T) {
	m := &manifestv1.ServerManifest{
		ServerID: "s1", Version: "1", GameVersion: "42.8",
		Mods: []manifestv1.ModEntry{
			{ID: "a", Version: "1", SHA256: "4fbc716b086d0746df8f5c7c04064ef6b0ef30ec8c2b9ea6ff9d7c8222fcb0fd", WorkshopID: "1", Dependencies: []string{"b"}},
			{ID: "b", Version: "1", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", WorkshopID: "2", Dependencies: []string{"a"}},
		},
	}
	_, err := FromManifest(m)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
