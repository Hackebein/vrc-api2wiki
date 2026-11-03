package steam

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSessionHome(t *testing.T) {
	dd := filepath.Join("/repo", "third_party", "DepotDownloader", "DepotDownloader")
	got := SessionHome(dd)
	want := filepath.Join("/repo", ".steam-session")
	if got != want {
		t.Fatalf("SessionHome = %q, want %q", got, want)
	}
}

func TestEnvWithHomeOverrides(t *testing.T) {
	t.Setenv("HOME", "/original/home")
	t.Setenv("XDG_DATA_HOME", "/outside/the/cached/session")
	env := envWithHome("/tmp/steam-home")
	var home, profile, dataHome string
	homes, profiles, dataHomes := 0, 0, 0
	for _, e := range env {
		switch {
		case strings.HasPrefix(e, "HOME="):
			homes++
			home = strings.TrimPrefix(e, "HOME=")
		case strings.HasPrefix(e, "USERPROFILE="):
			profiles++
			profile = strings.TrimPrefix(e, "USERPROFILE=")
		case strings.HasPrefix(e, "XDG_DATA_HOME="):
			dataHomes++
			dataHome = strings.TrimPrefix(e, "XDG_DATA_HOME=")
		}
	}
	if homes != 1 || home != "/tmp/steam-home" {
		t.Fatalf("HOME count=%d value=%q", homes, home)
	}
	if want := filepath.Join("/tmp/steam-home", ".local", "share"); dataHomes != 1 || dataHome != want {
		t.Fatalf("XDG_DATA_HOME count=%d value=%q, want %q", dataHomes, dataHome, want)
	}
	if runtime.GOOS == "windows" {
		if profiles != 1 || profile != "/tmp/steam-home" {
			t.Fatalf("USERPROFILE count=%d value=%q", profiles, profile)
		}
	}
}
