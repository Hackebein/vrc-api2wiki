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
	env := envWithHome("/tmp/steam-home")
	var home, profile string
	homes, profiles := 0, 0
	for _, e := range env {
		switch {
		case strings.HasPrefix(e, "HOME="):
			homes++
			home = strings.TrimPrefix(e, "HOME=")
		case strings.HasPrefix(e, "USERPROFILE="):
			profiles++
			profile = strings.TrimPrefix(e, "USERPROFILE=")
		}
	}
	if homes != 1 || home != "/tmp/steam-home" {
		t.Fatalf("HOME count=%d value=%q", homes, home)
	}
	if runtime.GOOS == "windows" {
		if profiles != 1 || profile != "/tmp/steam-home" {
			t.Fatalf("USERPROFILE count=%d value=%q", profiles, profile)
		}
	}
}
