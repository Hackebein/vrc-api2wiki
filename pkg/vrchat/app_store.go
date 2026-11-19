package vrchat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
)

const appStoreID = 6448578770

type AppStoreRelease struct {
	Version   string
	LookupURL string
	StoreURL  string
}

// FetchAppStoreRelease selects the newest observed iOS/iPadOS release across
// the US and German storefronts. Both reads must succeed before using the result.
func FetchAppStoreRelease(client *http.Client) (AppStoreRelease, error) {
	var newest AppStoreRelease
	var newestVersion [4]int
	for _, country := range []string{"us", "de"} {
		release, err := lookupAppStoreRelease(client, country)
		if err != nil {
			return AppStoreRelease{}, fmt.Errorf("App Store %s: %w", country, err)
		}
		version, ok := parseReleaseVersion(release.Version)
		if !ok {
			return AppStoreRelease{}, fmt.Errorf("App Store %s: unexpected VRChat version %q", country, release.Version)
		}
		if slices.Compare(version[:], newestVersion[:]) > 0 {
			newest, newestVersion = release, version
		}
	}
	return newest, nil
}

func lookupAppStoreRelease(client *http.Client, country string) (AppStoreRelease, error) {
	lookupURL := fmt.Sprintf("https://itunes.apple.com/lookup?id=%d&country=%s", appStoreID, country)
	resp, err := client.Get(lookupURL)
	if err != nil {
		return AppStoreRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return AppStoreRelease{}, fmt.Errorf("lookup returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		Results []struct {
			TrackID  int    `json:"trackId"`
			BundleID string `json:"bundleId"`
			Version  string `json:"version"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return AppStoreRelease{}, fmt.Errorf("decode lookup: %w", err)
	}
	for _, app := range result.Results {
		if app.TrackID == appStoreID && app.BundleID == "com.vrchat.mobile" {
			return AppStoreRelease{
				Version:   app.Version,
				LookupURL: lookupURL,
				StoreURL:  fmt.Sprintf("https://apps.apple.com/%s/app/vrchat/id%d", country, appStoreID),
			}, nil
		}
	}
	return AppStoreRelease{}, fmt.Errorf("no matching VRChat listing")
}
