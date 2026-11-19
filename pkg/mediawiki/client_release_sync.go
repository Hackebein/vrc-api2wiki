package mediawiki

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

const (
	iosClientName   = "ios"
	mainReleaseName = "live-mainpage"
)

func syncIOSRelease(wiki *MediaWikiClient, root string, client *http.Client, logger *slog.Logger) error {
	listing, err := vrchat.FetchAppStoreRelease(client)
	if err != nil {
		if logger != nil {
			logger.Warn("App Store listing unavailable", "err", err)
		}
		return nil
	}
	release := &steam.ClientBuild{
		Version: listing.Version, SourceURL: listing.LookupURL, ReleaseNotesURL: listing.StoreURL,
		Branch: iosClientName, FetchedAt: time.Now().UTC(),
	}
	notes, err := vrchat.FetchIOSReleaseNotes(client, listing.Version)
	if err == nil {
		release.BuildNumber = notes.BuildNumber
		release.ReleaseNotesURL = notes.URL
	} else if !errors.Is(err, vrchat.ErrReleaseNotesNotFound) {
		if logger != nil {
			logger.Warn("iOS release notes lookup failed", "version", listing.Version, "err", err)
		}
		return nil
	}
	if release.BuildNumber == "" && logger != nil {
		logger.Warn("iOS store version has no matching published build number", "version", listing.Version)
	}
	return writeClientRelease(wiki, root, release, logger)
}

func syncLiveMainRelease(wiki *MediaWikiClient, root string, client *http.Client, logger *slog.Logger) error {
	notes, err := vrchat.FetchLiveMainRelease(client)
	if err != nil {
		if logger != nil {
			logger.Warn("main release metadata unavailable", "err", err)
		}
		return nil
	}
	return writeClientRelease(wiki, root, &steam.ClientBuild{
		Version: notes.Version, BuildNumber: notes.BuildNumber, ReleaseDate: notes.ReleaseDate,
		ReleaseNotesURL: notes.URL, SourceURL: vrchat.LatestLiveReleaseURL,
		Branch: mainReleaseName, FetchedAt: time.Now().UTC(),
	}, logger)
}

// Publish each release as one template revision so a failed edit cannot
// leave readers with a date or build number for a different version.
func writeClientRelease(wiki *MediaWikiClient, root string, release *steam.ClientBuild, logger *slog.Logger) error {
	if err := steam.WriteBuildJSON(filepath.Join(root, "steam-output", release.Branch+".json"), release); err != nil {
		return err
	}
	display := release.Version
	if release.BuildNumber != "" {
		display += " (" + release.BuildNumber + ")"
	}
	text := fmt.Sprintf(`{{#switch:{{{1|display}}}
 |version = %s
 |buildNumber = %s
 |releaseNotes = %s
 |releaseDate = %s
 |display = %s
 |#default =
}}`, release.Version, release.BuildNumber, release.ReleaseNotesURL, release.ReleaseDate, display)
	title := ClientBuildPageTitle(release.Branch, "")
	if err := wiki.EditPage(title, text, true); err != nil {
		return fmt.Errorf("edit %s: %w", title, err)
	}
	if logger != nil {
		logger.Info("client release written", "client", release.Branch, "version", release.Version,
			"buildNumber", release.BuildNumber, "releaseDate", release.ReleaseDate,
			"source", release.SourceURL, "releaseNotes", release.ReleaseNotesURL)
	}
	return nil
}
