package game

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildLaunchArgsSplitsFields(t *testing.T) {
	args := buildLaunchArgs("-debug -nosound", "")
	if len(args) != 2 || args[0] != "-debug" || args[1] != "-nosound" {
		t.Fatalf("expected [-debug -nosound] as separate argv elements, got %v", args)
	}
}

func TestBuildLaunchArgsInjectsCachedir(t *testing.T) {
	profile := t.TempDir()
	args := buildLaunchArgs("-debug", profile)

	abs, _ := filepath.Abs(profile)
	found := false
	for _, a := range args {
		if a == "-cachedir="+abs {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected -cachedir=%s in %v", abs, args)
	}
}

func TestBuildLaunchArgsRespectsUserCachedir(t *testing.T) {
	args := buildLaunchArgs(`-cachedir=D:\custom -debug`, t.TempDir())

	count := 0
	for _, a := range args {
		if strings.HasPrefix(a, "-cachedir") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one -cachedir (user's), got %v", args)
	}
}

func TestBuildLaunchArgsEmpty(t *testing.T) {
	if args := buildLaunchArgs("", ""); len(args) != 0 {
		t.Fatalf("expected no args, got %v", args)
	}
}
