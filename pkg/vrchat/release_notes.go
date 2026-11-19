package vrchat

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	LatestLiveReleaseURL = "https://docs.vrchat.com/docs/latest-release"
	releaseIndexURL      = "https://docs.vrchat.com/llms.txt"
	versionSyntax        = `[1-9][0-9]{3}\.[0-9]{1,3}\.[0-9]{1,3}(?:p[1-9][0-9]{0,3})?`
	releaseURLSyntax     = `https://docs\.vrchat\.com/docs/[a-z0-9-]+`
)

var (
	ErrReleaseNotesNotFound = errors.New("no matching client release notes")
	releaseVersionPattern   = regexp.MustCompile(`^` + versionSyntax + `$`)
	releaseURLPattern       = regexp.MustCompile(`^` + releaseURLSyntax + `$`)
	releaseIndexLine        = regexp.MustCompile(`(?m)^[ \t]*- \[VRChat (` + versionSyntax + `)(?: \(([^)\r\n]+)\))?\]\((` + releaseURLSyntax + `)\.md\)`)
	releaseHeading          = regexp.MustCompile(`(?m)^# VRChat (` + versionSyntax + `)(?: \(([^)\r\n]+)\))?[ \t]*\r?$`)
	nextHeading             = regexp.MustCompile(`(?m)^#{1,6}[ \t]`)
	datedBuildLine          = regexp.MustCompile("(?im)^([^\r\n]+?) +[-–—] +`?Build +`?(\\d{1,9})`? +[-–—] +(?:Live|Release)[ \\t]*\\r?$")
	inlineBuild             = regexp.MustCompile("(?i)`?(" + versionSyntax + ")`? +build +`?(\\d{1,9})`?\\b")
)

type ClientReleaseNotes struct {
	Version     string
	BuildNumber string
	ReleaseDate string // ISO 8601 date from the live release heading.
	URL         string
}

func parseReleaseVersion(version string) ([4]int, bool) {
	var parts [4]int
	if !releaseVersionPattern.MatchString(version) {
		return parts, false
	}
	for i, part := range strings.FieldsFunc(version, func(r rune) bool { return r == '.' || r == 'p' }) {
		parts[i], _ = strconv.Atoi(part)
	}
	return parts, true
}

// FetchIOSReleaseNotes matches the exact App Store version to VRChat's public
// release notes. The returned number identifies the VRChat release.
func FetchIOSReleaseNotes(client *http.Client, version string) (ClientReleaseNotes, error) {
	return fetchIndexedReleaseNotes(client, version, true)
}

// FetchLiveMainRelease resolves the official latest-release link. If the link
// points to a hotfix, retrieve that release family's original live notes.
func FetchLiveMainRelease(client *http.Client) (ClientReleaseNotes, error) {
	resp, err := client.Get(LatestLiveReleaseURL)
	if err != nil {
		return ClientReleaseNotes{}, err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ClientReleaseNotes{}, fmt.Errorf("latest release returned HTTP %d", resp.StatusCode)
	}
	target := resp.Request.URL.String()
	if target == LatestLiveReleaseURL || !releaseURLPattern.MatchString(target) {
		return ClientReleaseNotes{}, fmt.Errorf("latest release did not resolve to official release notes")
	}
	notes, err := fetchReleaseNotes(client, target, "", false)
	if err != nil {
		return ClientReleaseNotes{}, err
	}
	if base, _, patched := strings.Cut(notes.Version, "p"); patched {
		notes, err = fetchIndexedReleaseNotes(client, base, false)
		if err != nil {
			return ClientReleaseNotes{}, err
		}
	}
	if notes.BuildNumber == "" || notes.ReleaseDate == "" {
		return ClientReleaseNotes{}, fmt.Errorf("main release %s is missing a build number or live release date", notes.Version)
	}
	return notes, nil
}

func fetchIndexedReleaseNotes(client *http.Client, version string, ios bool) (ClientReleaseNotes, error) {
	index, err := fetchReleaseMarkdown(client, releaseIndexURL)
	if err != nil {
		return ClientReleaseNotes{}, err
	}
	entries := releaseIndexLine.FindAllStringSubmatch(index, -1)
	if len(entries) == 0 {
		return ClientReleaseNotes{}, fmt.Errorf("VRChat documentation index contains no client releases")
	}
	var candidate string
	for _, entry := range entries {
		if entry[1] != version {
			continue
		}
		if ios && isIOSRelease(entry[2]) {
			candidate = entry[3]
			break
		}
		if entry[2] == "" {
			candidate = entry[3]
		}
	}
	if candidate == "" {
		return ClientReleaseNotes{}, fmt.Errorf("%w for %s", ErrReleaseNotesNotFound, version)
	}
	return fetchReleaseNotes(client, candidate, version, ios)
}

func fetchReleaseNotes(client *http.Client, target, expectedVersion string, ios bool) (ClientReleaseNotes, error) {
	markdown, err := fetchReleaseMarkdown(client, target+".md")
	if err != nil {
		return ClientReleaseNotes{}, err
	}
	title := releaseHeading.FindStringSubmatchIndex(markdown)
	if title == nil {
		return ClientReleaseNotes{}, fmt.Errorf("no VRChat release heading at %s", target)
	}
	version := markdown[title[2]:title[3]]
	if expectedVersion != "" && version != expectedVersion {
		return ClientReleaseNotes{}, fmt.Errorf("release notes at %s show %s, expected %s", target, version, expectedVersion)
	}
	if title[4] >= 0 && !(ios && isIOSRelease(markdown[title[4]:title[5]])) {
		return ClientReleaseNotes{}, fmt.Errorf("release notes at %s are for a different platform or channel", target)
	}

	notes := ClientReleaseNotes{Version: version, URL: target}
	intro := markdown[title[1]:]
	if section := nextHeading.FindStringIndex(intro); section != nil {
		intro = intro[:section[0]]
	}
	if line := datedBuildLine.FindStringSubmatch(intro); line != nil {
		notes.BuildNumber = line[2]
		date := strings.TrimSpace(line[1])
		if !strings.Contains(date, ",") {
			date += ", " + version[:4]
		}
		if parsed, err := time.Parse("January 2, 2006", date); err == nil {
			notes.ReleaseDate = parsed.Format(time.DateOnly)
		}
	} else {
		// Some hotfix notes name the version and build together in prose.
		for _, match := range inlineBuild.FindAllStringSubmatch(intro, -1) {
			if match[1] == version {
				if notes.BuildNumber != "" && notes.BuildNumber != match[2] {
					return ClientReleaseNotes{}, fmt.Errorf("conflicting build numbers in release notes for %s", version)
				}
				notes.BuildNumber = match[2]
			}
		}
	}
	return notes, nil
}

func isIOSRelease(qualifier string) bool {
	return strings.EqualFold(qualifier, "iOS") || strings.EqualFold(qualifier, "iOS Only")
}

func fetchReleaseMarkdown(client *http.Client, source string) (string, error) {
	resp, err := client.Get(source)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s returned HTTP %d", source, resp.StatusCode)
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/") || strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		return "", fmt.Errorf("unexpected document type from %s", source)
	}
	const maxBytes = 2 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxBytes {
		return "", fmt.Errorf("release document at %s exceeded %d bytes", source, maxBytes)
	}
	return string(data), nil
}
