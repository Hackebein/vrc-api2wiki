// check-steam-session verifies Steam authentication without writing to the wiki.
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

func main() {
	root, err := steam.FindRepoRoot()
	if err != nil {
		log.Fatal(err)
	}
	username := os.Getenv("STEAM_USERNAME")
	if username == "" {
		log.Fatal("STEAM_USERNAME is required")
	}
	password, shared := os.Getenv("STEAM_PASSWORD"), os.Getenv("STEAM_SHARED_SECRET")
	reuseOnly := os.Getenv("REQUIRE_STEAM_SESSION_REUSE") == "true"
	if reuseOnly {
		password, shared = "", ""
	} else if password == "" || shared == "" {
		log.Fatal("STEAM_PASSWORD and STEAM_SHARED_SECRET are required to bootstrap a session")
	}
	// An empty file list authenticates and fetches depot metadata without
	// downloading the client. Supplying no credentials proves token reuse.
	_, _, err = steam.DownloadDepot(steam.DepotDownloaderPath(root), steam.VRChatWindowsDepotID,
		"public", filepath.Join(root, "steam-output", "session-check"), username, password, shared, nil)
	if err != nil {
		log.Fatal(err)
	}
	if reuseOnly {
		log.Println("Steam session reuse verified without password or Steam Guard code")
	} else {
		log.Println("Steam session is ready for a token-only check on a fresh runner")
	}
}
