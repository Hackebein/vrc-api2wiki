package main

import (
	"log"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/mediawiki"
	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	stdLogger := log.New(os.Stdout, "vrc-api2wiki ", log.LstdFlags)

	wikiAPI := os.Getenv("VRCWIKI_API_URL")
	if wikiAPI == "" {
		wikiAPI = "https://wiki.vrchat.com/api.php"
	}
	wikiUser := os.Getenv("VRCWIKI_USERNAME")
	wikiPass := os.Getenv("VRCWIKI_PASSWORD")
	wikiHdrName := os.Getenv("VRCWIKI_AUTHORIZATION_HEADER")
	wikiHdrValue := os.Getenv("VRCWIKI_AUTHORIZATION_VALUE")

	jar, err := cookiejar.New(nil)
	if err != nil {
		stdLogger.Fatalf("cookie jar: %v", err)
	}
	httpClient := &http.Client{Timeout: 120 * time.Second, Jar: jar}

	wikiClient, err := mediawiki.NewMediaWikiClient(mediawiki.WikiConfig{
		URL:       wikiAPI,
		Username:  wikiUser,
		Password:  wikiPass,
		Header:    wikiHdrName,
		HeaderVal: wikiHdrValue,
	}, httpClient)
	if err != nil {
		stdLogger.Fatalf("init wiki client: %v", err)
	}

	apiClient := vrchat.NewClient(httpClient)

	stdLogger.Println("running world sync")
	if err := mediawiki.RunSync(wikiClient, apiClient, logger); err != nil {
		stdLogger.Fatalf("world sync failed: %v", err)
	}
	stdLogger.Println("world sync complete")

	stdLogger.Println("running marketplace sync")
	if err := mediawiki.RunStoreSync(wikiClient, apiClient, logger); err != nil {
		stdLogger.Fatalf("marketplace sync failed: %v", err)
	}
	stdLogger.Println("marketplace sync complete")

	stdLogger.Println("running steam sync")
	if err := mediawiki.RunSteamSync(wikiClient, logger); err != nil {
		stdLogger.Fatalf("steam sync failed: %v", err)
	}
	stdLogger.Println("steam sync complete")
}
