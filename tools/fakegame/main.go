// fakegame stands in for the Project Zomboid client in test labs.
// It records the argv it was launched with (launch-args.txt next to the
// executable) and exits after a short delay, so lab verify scripts can assert
// the launcher passed profile isolation flags without installing the game.
//
// Build as ProjectZomboid64[.exe] into a directory and point PZ_GAME_PATH at it.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out := filepath.Join(filepath.Dir(exe), "launch-args.txt")
	if err := os.WriteFile(out, []byte(strings.Join(os.Args[1:], "\n")+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	time.Sleep(2 * time.Second)
}
