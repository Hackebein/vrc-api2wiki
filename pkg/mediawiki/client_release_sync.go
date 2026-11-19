package mediawiki

import (
	"errors"
	"log/slog"
	"net/http"
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
	return publishClientBuild(wiki, root, release, logger)
}

func syncLiveMainRelease(wiki *MediaWikiClient, root string, client *http.Client, logger *slog.Logger) error {
	notes, err := vrchat.FetchLiveMainRelease(client)
	if err != nil {
		if logger != nil {
			logger.Warn("main release metadata unavailable", "err", err)
		}
		return nil
	}
	return publishClientBuild(wiki, root, &steam.ClientBuild{
		Version: notes.Version, BuildNumber: notes.BuildNumber, ReleaseDate: notes.ReleaseDate,
		ReleaseNotesURL: notes.URL, SourceURL: vrchat.LatestLiveReleaseURL,
		Branch: mainReleaseName, FetchedAt: time.Now().UTC(),
	}, logger)
}
