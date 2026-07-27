package steam

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ClientBuild struct {
	Version      string    `json:"version"`
	BuildNumber  string    `json:"buildNumber"`
	BuildHash    string    `json:"buildHash"`
	SteamBuildID string    `json:"steamBuildId,omitempty"`
	ManifestID   string    `json:"manifestId,omitempty"`
	Branch       string    `json:"branch"`
	FetchedAt    time.Time `json:"fetchedAt"`
	RawMatch     string    `json:"rawMatch,omitempty"`
}

var buildPattern = regexp.MustCompile(`(\d{4}\.\d+\.\d+[a-zA-Z]?\d*)-(\d+)-([0-9a-fA-F]{7,40})(?:-(?:Release|OpenBeta|Beta))?`)

func ExtractBuildFromBytes(data []byte, branch string) (*ClientBuild, error) {
	matches := buildPattern.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no VRChat build string found")
	}
	return clientBuildFromMatch(matches[len(matches)-1], branch), nil
}

// ExtractHighestBuildFromBytes picks the match with the largest numeric build number.
// Useful when a page embeds multiple historical version strings.
func ExtractHighestBuildFromBytes(data []byte, branch string) (*ClientBuild, error) {
	matches := buildPattern.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no VRChat build string found")
	}
	bestIdx := 0
	bestNum := -1
	for i, m := range matches {
		n, err := strconv.Atoi(string(m[2]))
		if err != nil {
			continue
		}
		if n >= bestNum {
			bestNum = n
			bestIdx = i
		}
	}
	if bestNum < 0 {
		return clientBuildFromMatch(matches[len(matches)-1], branch), nil
	}
	return clientBuildFromMatch(matches[bestIdx], branch), nil
}

func clientBuildFromMatch(m [][]byte, branch string) *ClientBuild {
	return &ClientBuild{
		Version:     string(m[1]),
		BuildNumber: string(m[2]),
		BuildHash:   strings.ToLower(string(m[3])),
		Branch:      branch,
		FetchedAt:   time.Now().UTC(),
		RawMatch:    string(m[0]),
	}
}

func ExtractBuildFromDir(dir, branch string) (*ClientBuild, error) {
	candidates := []string{
		"VRChat_Data/globalgamemanagers",
		"VRChat.exe",
		"UnityPlayer.dll",
		"VRChat_Data/Managed/Assembly-CSharp.dll",
	}
	var lastErr error
	for _, rel := range candidates {
		path := filepath.Join(dir, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			if lastErr == nil {
				lastErr = err
			}
			continue
		}
		cb, err := ExtractBuildFromBytes(data, branch)
		if err != nil {
			lastErr = err
			continue
		}
		return cb, nil
	}

	var found *ClientBuild
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if found != nil || err != nil || info.IsDir() {
			return nil
		}
		if info.Size() == 0 || info.Size() > 64<<20 {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		cb, rerr := ExtractBuildFromBytes(data, branch)
		if rerr != nil {
			return nil
		}
		found = cb
		return filepath.SkipAll
	})
	if found != nil {
		return found, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("extract build from %s: %w", dir, lastErr)
	}
	return nil, fmt.Errorf("extract build from %s: no candidates", dir)
}

func WriteBuildJSON(path string, build *ClientBuild) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(build, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func ClientBuildPages(b *ClientBuild) map[string]string {
	return map[string]string{
		"version":     b.Version,
		"buildNumber": b.BuildNumber,
		"buildHash":   b.BuildHash,
	}
}
